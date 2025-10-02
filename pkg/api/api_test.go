package api

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type APITestSuite struct {
	suite.Suite
	api *API
}

func TestAPITestSuite(t *testing.T) {
	suite.Run(t, &APITestSuite{
		api: NewAPI("3.0.0", "en"),
	})
}

func (s *APITestSuite) TestGetLatestVersion() {
	_, err := s.api.LatestVersion("stable")
	s.NoError(err)
}

func (s *APITestSuite) TestGetIntermediateVersions() {
	_, err := s.api.IntermediateVersions("stable")
	s.NoError(err)
}

func (s *APITestSuite) TestGetCategories() {
	_, err := s.api.Categories()
	s.NoError(err)
}

func (s *APITestSuite) TestGetApps() {
	_, err := s.api.Apps()
	s.NoError(err)
}

func (s *APITestSuite) TestGetAppBySlug() {
	_, err := s.api.AppBySlug("nginx")
	s.NoError(err)
}

func (s *APITestSuite) TestAppCallback() {
	err := s.api.AppCallback("nginx")
	s.NoError(err)
}

func (s *APITestSuite) TestGetTemplates() {
	_, err := s.api.Templates()
	s.NoError(err)
}

func (s *APITestSuite) TestGetTemplateBySlug() {
	_, err := s.api.TemplateBySlug("nginx")
	s.NoError(err)
}

func (s *APITestSuite) TestTemplateCallback() {
	err := s.api.TemplateCallback("nginx")
	s.NoError(err)
}

func (s *APITestSuite) TestGetRewritesByType() {
	_, err := s.api.RewritesByType("nginx")
	s.NoError(err)
}
