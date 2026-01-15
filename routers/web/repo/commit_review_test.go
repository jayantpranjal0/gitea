// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"testing"

	git_model "code.gitea.io/gitea/models/git"
	"code.gitea.io/gitea/models/unittest"
	"code.gitea.io/gitea/modules/templates"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/services/contexttest"
	"code.gitea.io/gitea/services/forms"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderNewCommitCommentForm(t *testing.T) {
	unittest.PrepareTestEnv(t)

	ctx, resp := contexttest.MockContext(t, "/", contexttest.MockContextOption{Render: templates.HTMLRenderer()})
	contexttest.LoadUser(t, ctx, 1)
	contexttest.LoadRepo(t, ctx, 1)
	contexttest.LoadGitRepo(t, ctx)
	defer ctx.Repo.GitRepo.Close()

	// set a commit id
	sha, err := ctx.Repo.GitRepo.GetBranchCommitID("master")
	require.NoError(t, err)
	ctx.SetPathParam("sha", sha)

	RenderNewCommitCommentForm(ctx)
	assert.Equal(t, 200, resp.Code)
	assert.Contains(t, resp.Body.String(), "textarea")
}

func TestCreateCommitCodeComment(t *testing.T) {
	unittest.PrepareTestEnv(t)

	ctx, resp := contexttest.MockContext(t, "/", contexttest.MockContextOption{Render: templates.HTMLRenderer()})
	contexttest.LoadUser(t, ctx, 1)
	contexttest.LoadRepo(t, ctx, 1)
	contexttest.LoadGitRepo(t, ctx)
	defer ctx.Repo.GitRepo.Close()

	sha, err := ctx.Repo.GitRepo.GetBranchCommitID("master")
	require.NoError(t, err)
	ctx.SetPathParam("sha", sha)

	// prepare form
	form := &forms.CodeCommentForm{
		Content:        "a comment on commit",
		Side:           "proposed",
		Line:           1,
		TreePath:       "README.md",
		LatestCommitID: sha,
	}
	// bind form to context
	web.SetForm(ctx, form)
	// set origin to diff so validation accepts it
	ctx.SetFormString("origin", "diff")

	// call handler
	CreateCommitCodeComment(ctx)
	assert.Equal(t, 200, resp.Code)
	// ensure response contains the comment content
	assert.Contains(t, resp.Body.String(), "a comment on commit")

	// check DB
	c := &git_model.CommitComment{CommitSHA: sha}
	exists := unittest.AssertExistsAndLoadBean(t, c)
	assert.NotNil(t, exists)
}

func TestCommitDiffContainsAddCodeComment(t *testing.T) {
	unittest.PrepareTestEnv(t)

	ctx, resp := contexttest.MockContext(t, "/", contexttest.MockContextOption{Render: templates.HTMLRenderer()})
	contexttest.LoadUser(t, ctx, 1)
	contexttest.LoadRepo(t, ctx, 1)
	contexttest.LoadGitRepo(t, ctx)
	defer ctx.Repo.GitRepo.Close()

	sha, err := ctx.Repo.GitRepo.GetBranchCommitID("master")
	require.NoError(t, err)
	ctx.SetPathParam("sha", sha)

	Diff(ctx)
	assert.Equal(t, 200, resp.Code)
	body := resp.Body.String()
	// data-new-comment-url should be present on the file table
	assert.Contains(t, body, `data-new-comment-url`)
	// add-code-comment button should be rendered (even if hidden via tw-invisible)
	assert.Contains(t, body, `add-code-comment`)
}
