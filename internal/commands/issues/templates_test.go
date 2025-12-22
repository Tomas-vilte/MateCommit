package issues

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/thomas-vilte/matecommit/internal/models"
	"github.com/urfave/cli/v3"
)

func TestIssueTemplateAction(t *testing.T) {
	t.Run("should init templates successfully", func(t *testing.T) {
		_, mockTemp, provider, trans, cfg := setupIssuesTest(t)
		factory := NewIssuesCommandFactory(provider, mockTemp)
		cmd := factory.CreateCommand(trans, cfg)

		mockTemp.On("InitializeTemplates", mock.Anything, false).Return(nil)
		mockTemp.On("GetTemplatesDir", mock.Anything).Return("/path/to/templates", nil)

		app := &cli.Command{Name: "test", Commands: []*cli.Command{cmd}}
		err := app.Run(context.Background(), []string{"test", "issue", "template", "init"})

		assert.NoError(t, err)
		mockTemp.AssertExpectations(t)
	})

	t.Run("should init templates with force flag", func(t *testing.T) {
		_, mockTemp, provider, trans, cfg := setupIssuesTest(t)
		factory := NewIssuesCommandFactory(provider, mockTemp)
		cmd := factory.CreateCommand(trans, cfg)

		mockTemp.On("InitializeTemplates", mock.Anything, true).Return(nil)
		mockTemp.On("GetTemplatesDir", mock.Anything).Return("/path/to/templates", nil)

		app := &cli.Command{Name: "test", Commands: []*cli.Command{cmd}}
		err := app.Run(context.Background(), []string{"test", "issue", "template", "init", "--force"})

		assert.NoError(t, err)
		mockTemp.AssertExpectations(t)
	})

	t.Run("should list templates successfully", func(t *testing.T) {
		_, mockTemp, provider, trans, cfg := setupIssuesTest(t)
		factory := NewIssuesCommandFactory(provider, mockTemp)
		cmd := factory.CreateCommand(trans, cfg)

		templates := []models.TemplateMetadata{
			{Name: "Bug", About: "Bug report", FilePath: "bug.md"},
			{Name: "Feature", About: "Feature request", FilePath: "feat.md"},
		}
		mockTemp.On("ListTemplates", mock.Anything).Return(templates, nil)

		app := &cli.Command{Name: "test", Commands: []*cli.Command{cmd}}
		err := app.Run(context.Background(), []string{"test", "issue", "template", "list"})

		assert.NoError(t, err)
		mockTemp.AssertExpectations(t)
	})

	t.Run("should handle empty template list", func(t *testing.T) {
		_, mockTemp, provider, trans, cfg := setupIssuesTest(t)
		factory := NewIssuesCommandFactory(provider, mockTemp)
		cmd := factory.CreateCommand(trans, cfg)

		mockTemp.On("ListTemplates", mock.Anything).Return([]models.TemplateMetadata{}, nil)

		app := &cli.Command{Name: "test", Commands: []*cli.Command{cmd}}
		err := app.Run(context.Background(), []string{"test", "issue", "template", "list"})

		assert.NoError(t, err)
		mockTemp.AssertExpectations(t)
	})
}
