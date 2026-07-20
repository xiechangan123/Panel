package data

import (
	"testing"
	"time"

	"github.com/leonelquinteros/gotext"
	"github.com/libtnb/sqlite"
	"gorm.io/gorm"

	"github.com/acepanel/panel/v3/internal/biz"
)

func newFileShareRepoForTest(t *testing.T) *fileShareRepo {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&biz.FileShare{}); err != nil {
		t.Fatal(err)
	}
	return &fileShareRepo{t: gotext.NewLocale("", "en"), db: db}
}

func TestFileShareRepeatCreate(t *testing.T) {
	repo := newFileShareRepoForTest(t)

	a, err := repo.Create("/tmp/a.txt", 0, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	b, err := repo.Create("/tmp/a.txt", 3, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("repeat Create: %v", err)
	}
	if a.Token == b.Token || a.ID == b.ID {
		t.Fatalf("repeat share should create a new record: %+v vs %+v", a, b)
	}

	shares, err := repo.List()
	if err != nil || len(shares) != 2 {
		t.Fatalf("List: %v, len=%d", err, len(shares))
	}
}

func TestFileShareConsume(t *testing.T) {
	repo := newFileShareRepoForTest(t)

	share, err := repo.Create("/tmp/b.txt", 2, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}

	// 续传分块请求只校验不计数
	for range 5 {
		if _, err = repo.Consume(share.Token, false); err != nil {
			t.Fatalf("Consume without count: %v", err)
		}
	}

	// 计数两次后达到上限
	for range 2 {
		if _, err = repo.Consume(share.Token, true); err != nil {
			t.Fatalf("Consume with count: %v", err)
		}
	}
	if _, err = repo.Consume(share.Token, true); err == nil {
		t.Fatal("Consume should fail after limit reached")
	}
	if _, err = repo.Consume(share.Token, false); err == nil {
		t.Fatal("Consume without count should also fail after limit reached")
	}

	// 不存在的 token
	if _, err = repo.Consume("nonexistent", true); err == nil {
		t.Fatal("Consume should fail for unknown token")
	}
}

func TestFileShareExpired(t *testing.T) {
	repo := newFileShareRepoForTest(t)

	share, err := repo.Create("/tmp/c.txt", 0, time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repo.Consume(share.Token, true); err == nil {
		t.Fatal("Consume should fail for expired share")
	}

	count, err := repo.ClearExpired()
	if err != nil || count != 1 {
		t.Fatalf("ClearExpired: %v, count=%d", err, count)
	}
	shares, _ := repo.List()
	if len(shares) != 0 {
		t.Fatalf("expired share should be cleared, len=%d", len(shares))
	}
}
