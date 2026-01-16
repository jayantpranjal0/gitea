// SPDX-License-Identifier: MIT
package repo

import (
	"net/http"
	"path"

	git_model "code.gitea.io/gitea/models/git"
	renderhelper "code.gitea.io/gitea/models/renderhelper"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/markup/markdown"
	"code.gitea.io/gitea/modules/util"
	"code.gitea.io/gitea/services/context"
)

// DeleteCommitComment deletes a commit comment
func DeleteCommitComment(ctx *context.Context) {
	id := ctx.PathParamInt64("id")
	cc, err := git_model.GetCommitCommentByID(ctx, id)
	if err != nil {
		ctx.NotFoundOrServerError("GetCommitCommentByID", git_model.IsErrCommitCommentNotExist, err)
		return
	}

	if cc.RepoID != ctx.Repo.Repository.ID {
		ctx.NotFound(err)
		return
	}

	if !ctx.IsSigned || (ctx.Doer.ID != cc.PosterID && !ctx.Repo.CanWriteToBranch(ctx, ctx.Doer, ctx.Repo.BranchName)) {
		// allow deletion by poster or users who can write to the branch (repo maintainers)
		ctx.HTTPError(http.StatusForbidden)
		return
	}

	if err := git_model.DeleteCommitComment(ctx, id); err != nil {
		ctx.ServerError("DeleteCommitComment", err)
		return
	}

	log.Trace("Commit comment deleted: %d/%d", ctx.Repo.Repository.ID, id)
	ctx.Status(http.StatusOK)
}

// UpdateCommitComment updates commit comment content
func UpdateCommitComment(ctx *context.Context) {
	id := ctx.PathParamInt64("id")
	cc, err := git_model.GetCommitCommentByID(ctx, id)
	if err != nil {
		ctx.NotFoundOrServerError("GetCommitCommentByID", git_model.IsErrCommitCommentNotExist, err)
		return
	}

	if cc.RepoID != ctx.Repo.Repository.ID {
		ctx.NotFound(err)
		return
	}

	if !ctx.IsSigned || (ctx.Doer.ID != cc.PosterID && !ctx.Repo.CanWriteToBranch(ctx, ctx.Doer, ctx.Repo.BranchName)) {
		ctx.HTTPError(http.StatusForbidden)
		return
	}

	newContent := ctx.FormString("content")
	if newContent != cc.Content {
		oldContent := cc.Content
		cc.Content = newContent
		if err := git_model.UpdateCommitComment(ctx, cc); err != nil {
			ctx.ServerError("UpdateCommitComment", err)
			return
		}
		log.Trace("Commit comment updated: %d/%d", ctx.Repo.Repository.ID, id)
		_ = oldContent // reserved for potential audit/logging
	}

	// render updated content using markdown renderer so newlines and markdown are preserved
	rctx := renderhelper.NewRenderContextRepoComment(ctx, ctx.Repo.Repository, renderhelper.RepoCommentOptions{CurrentRefPath: path.Join("commit", util.PathEscapeSegments(cc.CommitSHA))})
	renderedHTML, err := markdown.RenderString(rctx, cc.Content)
	if err != nil {
		ctx.ServerError("RenderString", err)
		return
	}

	ctx.JSON(http.StatusOK, map[string]any{
		"content":        renderedHTML,
		"contentVersion": cc.ContentVersion(),
		"attachments":    "",
	})
}
