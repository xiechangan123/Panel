//go:build linux

package tamper

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/asm"
	"github.com/cilium/ebpf/btf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
	"github.com/cilium/ebpf/rlimit"
	"golang.org/x/sys/unix"
)

// 写打开标志
const (
	oWrOnly = 1
	oRdWr   = 2
)

// 事件结构体大小:inode(8)+dev(8)+pid(4)+op(4)+comm(16)
const eventStructSize = 40

// protKey 保护集合的键:设备号+inode 号唯一标识一个文件
// 仅用 inode 会在不同文件系统(如 tmpfs)间冲突,导致合法操作被误拦
type protKey struct {
	Ino uint64
	Dev uint64
}

// btfOffsets 从内核 BTF 查得的字段偏移
type btfOffsets struct {
	fFlags uint32 // file.f_flags
	fInode uint32 // file.f_inode
	dInode uint32 // dentry.d_inode
	iIno   uint32 // inode.i_ino
	iSb    uint32 // inode.i_sb
	sDev   uint32 // super_block.s_dev
}

// hookSpec 描述一个 LSM 拦截点
type hookSpec struct {
	hook   string // LSM hook 名,如 file_open
	param  string // 目标参数名,如 file / dentry / old_dentry
	param2 string // 可选的第二个 dentry 目标参数(可能为负 dentry,需判空)
	isFile bool   // 目标是 struct file(需查 f_flags 判写)还是 dentry
	op     Op     // 命中时上报的操作类型
}

// 四个稳定的拦截点:写打开、删除、重命名(含改名覆盖)、属性修改
// inode_rename 需同时检查 new_dentry,否则可用改名覆盖绕过保护
// inode_setattr 拦截 chmod/chown/truncate(截断不经过写打开)
var lsmHooks = []hookSpec{
	{hook: "file_open", param: "file", isFile: true, op: OpWrite},
	{hook: "inode_unlink", param: "dentry", op: OpUnlink},
	{hook: "inode_rename", param: "old_dentry", param2: "new_dentry", op: OpRename},
	{hook: "inode_setattr", param: "dentry", op: OpSetattr},
}

// ebpfEngine BPF-LSM 防篡改引擎
type ebpfEngine struct {
	protected *ebpf.Map
	eventsMap *ebpf.Map
	progs     []*ebpf.Program
	links     []link.Link
	reader    *ringbuf.Reader

	out    chan Event
	closed chan struct{}
	once   sync.Once
}

// fieldOffset 查询 struct 字段字节偏移
func fieldOffset(spec *btf.Spec, structName, fieldName string) (uint32, error) {
	var s *btf.Struct
	if err := spec.TypeByName(structName, &s); err != nil {
		return 0, fmt.Errorf("btf struct %s: %w", structName, err)
	}
	for _, m := range s.Members {
		if m.Name == fieldName {
			return m.Offset.Bytes(), nil
		}
	}
	return 0, fmt.Errorf("btf struct %s has no member %s", structName, fieldName)
}

// paramLayout 查询 bpf_lsm_<hook> 的参数个数与目标参数下标
// LSM BPF 程序 ctx 布局为 [arg0, arg1, ..., prevRet],故 prevRet 位于 ctx[nargs]
func paramLayout(spec *btf.Spec, hook, param string) (paramIdx, nargs int, err error) {
	var fn *btf.Func
	if err = spec.TypeByName("bpf_lsm_"+hook, &fn); err != nil {
		return 0, 0, fmt.Errorf("btf func bpf_lsm_%s: %w", hook, err)
	}
	proto, ok := fn.Type.(*btf.FuncProto)
	if !ok {
		return 0, 0, fmt.Errorf("bpf_lsm_%s is not a func proto", hook)
	}
	idx := -1
	for i, p := range proto.Params {
		if p.Name == param {
			idx = i
		}
	}
	if idx < 0 {
		return 0, 0, fmt.Errorf("bpf_lsm_%s has no param %s", hook, param)
	}
	return idx, len(proto.Params), nil
}

// buildProg 为一个拦截点生成 LSM 程序指令
// 寄存器约定:R6=ctx 备份(事件预留后复用为事件指针),R7=前序返回值,
// R8=设备号,R9=inode 号(R6-R9 跨 helper 调用保留)
func buildProg(h hookSpec, offs btfOffsets, paramIdx, paramIdx2, nargs int, protectedFD, eventsFD int) asm.Instructions {
	retOff := int16(nargs * 8)

	// lookup 从 R2(inode 指针)取 {ino, dev} 复合键查 protected map,结果在 R0
	// R9 = i_ino,R8 = i_sb->s_dev(零扩展),命中后供事件写入
	lookup := asm.Instructions{
		asm.LoadMem(asm.R9, asm.R2, int16(offs.iIno), asm.DWord), // i_ino
		asm.LoadMem(asm.R8, asm.R2, int16(offs.iSb), asm.DWord),  // i_sb
		asm.LoadMem(asm.R8, asm.R8, int16(offs.sDev), asm.Word),  // s_dev
		asm.StoreMem(asm.RFP, -16, asm.R9, asm.DWord),
		asm.StoreMem(asm.RFP, -8, asm.R8, asm.DWord),
		asm.LoadMapPtr(asm.R1, protectedFD),
		asm.Mov.Reg(asm.R2, asm.RFP),
		asm.Add.Imm(asm.R2, -16),
		asm.FnMapLookupElem.Call(),
	}

	insns := asm.Instructions{
		// ctx 备份到 R6(helper 调用会破坏 R1),R7 = 前序 LSM 返回值,非 0 透传
		asm.Mov.Reg(asm.R6, asm.R1),
		asm.LoadMem(asm.R7, asm.R6, retOff, asm.DWord),
		asm.JNE.Imm(asm.R7, 0, "deny_prev"),
	}

	// 取出被操作文件的 inode 指针到 R2
	if h.isFile {
		insns = append(insns,
			asm.LoadMem(asm.R2, asm.R6, int16(paramIdx*8), asm.DWord), // file
			asm.LoadMem(asm.R3, asm.R2, int16(offs.fFlags), asm.Word), // f_flags
			asm.And.Imm(asm.R3, oWrOnly|oRdWr),
			asm.JEq.Imm(asm.R3, 0, "allow"),                            // 非写打开放行
			asm.LoadMem(asm.R2, asm.R2, int16(offs.fInode), asm.DWord), // f_inode
		)
	} else {
		insns = append(insns,
			asm.LoadMem(asm.R2, asm.R6, int16(paramIdx*8), asm.DWord),  // dentry
			asm.LoadMem(asm.R2, asm.R2, int16(offs.dInode), asm.DWord), // d_inode
		)
	}

	insns = append(insns, lookup...)

	if paramIdx2 >= 0 {
		// 双目标(rename):目标 1 未命中时再查目标 2(改名覆盖的既有文件)
		// 目标 2 可能为负 dentry(无覆盖目标),d_inode 为空则放行
		insns = append(insns, asm.JNE.Imm(asm.R0, 0, "hit"))
		insns = append(insns,
			asm.LoadMem(asm.R2, asm.R6, int16(paramIdx2*8), asm.DWord),
			asm.JEq.Imm(asm.R2, 0, "allow"),
			asm.LoadMem(asm.R2, asm.R2, int16(offs.dInode), asm.DWord),
			asm.JEq.Imm(asm.R2, 0, "allow"),
		)
		insns = append(insns, lookup...)
	}

	insns = append(insns,
		asm.JEq.Imm(asm.R0, 0, "allow"), // 未命中放行

		// 命中:预留 ringbuf 事件
		asm.LoadMapPtr(asm.R1, eventsFD).WithSymbol("hit"),
		asm.Mov.Imm(asm.R2, eventStructSize),
		asm.Mov.Imm(asm.R3, 0),
		asm.FnRingbufReserve.Call(),
		asm.JEq.Imm(asm.R0, 0, "deny"), // 预留失败也拒绝
		asm.Mov.Reg(asm.R6, asm.R0),

		asm.StoreMem(asm.R6, 0, asm.R9, asm.DWord), // inode
		asm.StoreMem(asm.R6, 8, asm.R8, asm.DWord), // dev
		asm.FnGetCurrentPidTgid.Call(),
		asm.RSh.Imm(asm.R0, 32),
		asm.StoreMem(asm.R6, 16, asm.R0, asm.Word),      // pid
		asm.StoreImm(asm.R6, 20, int64(h.op), asm.Word), // op
		asm.Mov.Reg(asm.R1, asm.R6),
		asm.Add.Imm(asm.R1, 24),
		asm.Mov.Imm(asm.R2, 16),
		asm.FnGetCurrentComm.Call(), // comm
		asm.Mov.Reg(asm.R1, asm.R6),
		asm.Mov.Imm(asm.R2, 0),
		asm.FnRingbufSubmit.Call(),

		asm.Mov.Imm(asm.R0, -1).WithSymbol("deny"), // -EPERM
		asm.Return(),
		asm.Mov.Imm(asm.R0, 0).WithSymbol("allow"),
		asm.Return(),
		asm.Mov.Reg(asm.R0, asm.R7).WithSymbol("deny_prev"),
		asm.Return(),
	)

	return insns
}

// newEBPFEngine 构建并加载 BPF-LSM 引擎
func newEBPFEngine() (*ebpfEngine, error) {
	if err := rlimit.RemoveMemlock(); err != nil {
		return nil, fmt.Errorf("failed to remove memlock limit: %w", err)
	}

	spec, err := btf.LoadKernelSpec()
	if err != nil {
		return nil, fmt.Errorf("failed to load kernel BTF: %w", err)
	}

	var offs btfOffsets
	for _, f := range []struct {
		dst             *uint32
		structN, fieldN string
	}{
		{&offs.fFlags, "file", "f_flags"},
		{&offs.fInode, "file", "f_inode"},
		{&offs.dInode, "dentry", "d_inode"},
		{&offs.iIno, "inode", "i_ino"},
		{&offs.iSb, "inode", "i_sb"},
		{&offs.sDev, "super_block", "s_dev"},
	} {
		if *f.dst, err = fieldOffset(spec, f.structN, f.fieldN); err != nil {
			return nil, err
		}
	}

	// NO_PREALLOC 按需分配,否则百万级 MaxEntries 会预占约 80MB 内核内存
	protected, err := ebpf.NewMap(&ebpf.MapSpec{
		Type: ebpf.Hash, KeySize: 16, ValueSize: 1, MaxEntries: 1 << 20,
		Flags: unix.BPF_F_NO_PREALLOC,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create protected map: %w", err)
	}
	events, err := ebpf.NewMap(&ebpf.MapSpec{Type: ebpf.RingBuf, MaxEntries: 1 << 20})
	if err != nil {
		protected.Close()
		return nil, fmt.Errorf("failed to create events ringbuf: %w", err)
	}

	e := &ebpfEngine{
		protected: protected,
		eventsMap: events,
		out:       make(chan Event, 256),
		closed:    make(chan struct{}),
	}

	// 逐个加载并 attach LSM 程序
	for _, h := range lsmHooks {
		paramIdx, nargs, err := paramLayout(spec, h.hook, h.param)
		if err != nil {
			e.Close()
			return nil, err
		}
		paramIdx2 := -1
		if h.param2 != "" {
			if paramIdx2, _, err = paramLayout(spec, h.hook, h.param2); err != nil {
				e.Close()
				return nil, err
			}
		}
		prog, err := ebpf.NewProgram(&ebpf.ProgramSpec{
			Name:         "tamper_" + h.hook,
			Type:         ebpf.LSM,
			AttachType:   ebpf.AttachLSMMac,
			AttachTo:     h.hook,
			License:      "GPL",
			Instructions: buildProg(h, offs, paramIdx, paramIdx2, nargs, protected.FD(), events.FD()),
		})
		if err != nil {
			e.Close()
			if ve, ok := errors.AsType[*ebpf.VerifierError](err); ok {
				return nil, fmt.Errorf("failed to load %s LSM program: %+v", h.hook, ve)
			}
			return nil, fmt.Errorf("failed to load %s LSM program: %w", h.hook, err)
		}
		e.progs = append(e.progs, prog)

		l, err := link.AttachLSM(link.LSMOptions{Program: prog})
		if err != nil {
			e.Close()
			return nil, fmt.Errorf("failed to attach %s: %w", h.hook, err)
		}
		e.links = append(e.links, l)
	}

	e.reader, err = ringbuf.NewReader(events)
	if err != nil {
		e.Close()
		return nil, fmt.Errorf("failed to create ringbuf reader: %w", err)
	}

	go e.readLoop()
	return e, nil
}

// readLoop 从 ringbuf 读取拦截事件
func (e *ebpfEngine) readLoop() {
	for {
		rec, err := e.reader.Read()
		if err != nil {
			return // reader 关闭
		}
		b := rec.RawSample
		if len(b) < eventStructSize {
			continue
		}
		// BPF 按主机字节序写入
		op := Op(binary.NativeEndian.Uint32(b[20:24]))
		ev := Event{
			Inode: binary.NativeEndian.Uint64(b[0:8]),
			Dev:   binary.NativeEndian.Uint64(b[8:16]),
			PID:   binary.NativeEndian.Uint32(b[16:20]),
			Op:    op,
			OpStr: op.String(),
			Comm:  string(bytes.TrimRight(b[24:40], "\x00")),
		}
		select {
		case e.out <- ev:
		case <-e.closed:
			return
		default: // 消费不及时则丢弃,避免阻塞内核事件
		}
	}
}

// keysOf 从条目提取受保护文件的复合键(目录对 eBPF 无意义,跳过)
func keysOf(entries []fileEntry) []protKey {
	keys := make([]protKey, 0, len(entries))
	for _, en := range entries {
		if !en.isDir && en.inode != 0 {
			keys = append(keys, protKey{Ino: en.inode, Dev: en.dev})
		}
	}
	return keys
}

// apply 将条目中的文件加入保护集合
func (e *ebpfEngine) apply(entries []fileEntry) error {
	return e.Add(keysOf(entries))
}

// remove 将条目中的文件移出保护集合
func (e *ebpfEngine) remove(entries []fileEntry) error {
	return e.Remove(keysOf(entries))
}

func (e *ebpfEngine) events() <-chan Event { return e.out }

func (e *ebpfEngine) close() error { return e.Close() }

// Add 将文件加入保护集合(批量更新,避免大目录树逐键 syscall)
func (e *ebpfEngine) Add(keys []protKey) error {
	if len(keys) == 0 {
		return nil
	}
	values := make([]uint8, len(keys))
	for i := range values {
		values[i] = 1
	}
	_, err := e.protected.BatchUpdate(keys, values, nil)
	return err
}

// Remove 将文件移出保护集合
func (e *ebpfEngine) Remove(keys []protKey) error {
	for _, k := range keys {
		_ = e.protected.Delete(k)
	}
	return nil
}

// Close 释放所有资源
func (e *ebpfEngine) Close() error {
	e.once.Do(func() { close(e.closed) })
	if e.reader != nil {
		_ = e.reader.Close()
	}
	for _, l := range e.links {
		_ = l.Close()
	}
	for _, p := range e.progs {
		_ = p.Close()
	}
	if e.eventsMap != nil {
		_ = e.eventsMap.Close()
	}
	if e.protected != nil {
		_ = e.protected.Close()
	}
	return nil
}
