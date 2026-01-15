// Copyright 2026 The Gitea Authors.
// SPDX-License-Identifier: MIT

package git

import (
	"context"
	"fmt"
	"html/template"

	"code.gitea.io/gitea/models/db"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/timeutil"

	"code.gitea.io/gitea/modules/log"
)

// CommitComment represents a comment on a single commit
type CommitComment struct {
	ID              int64              `xorm:"pk autoincr"`
	RepoID          int64              `xorm:"INDEX"`
	CommitSHA       string             `xorm:"VARCHAR(64) INDEX"`
	PosterID        int64              `xorm:"INDEX"`
	Poster          *user_model.User   `xorm:"-"`
	Path            string             `xorm:"VARCHAR(4000)"`
	Line            int64              `xorm:"INDEX"`
	Content         string             `xorm:"LONGTEXT"`
	RenderedContent template.HTML      `xorm:"-"`
	CreatedUnix     timeutil.TimeStamp `xorm:"INDEX created"`
	UpdatedUnix     timeutil.TimeStamp `xorm:"INDEX updated"`
}

func init() {
	db.RegisterModel(new(CommitComment))
}

// CreateCommitComment inserts a new commit comment
func CreateCommitComment(ctx context.Context, c *CommitComment) error {
	if c == nil {
		return fmt.Errorf("nil commit comment")
	}
	if _, err := db.GetEngine(ctx).Insert(c); err != nil {
		return fmt.Errorf("Insert CommitComment: %w", err)
	}
	return nil
}

// GetCommitCommentByID returns a commit comment by id
func GetCommitCommentByID(ctx context.Context, id int64) (*CommitComment, error) {
	c := &CommitComment{ID: id}
	has, err := db.GetEngine(ctx).ID(id).Get(c)
	if err != nil {
		return nil, fmt.Errorf("Get CommitComment by id: %w", err)
	}
	if !has {
		return nil, fmt.Errorf("commit comment does not exist [id: %d]", id)
	}
	return c, nil
}

// ListCommitComments returns commit comments for a repo and commit sha
func ListCommitComments(ctx context.Context, repoID int64, commitSHA string) ([]*CommitComment, error) {
	var cs []*CommitComment
	err := db.GetEngine(ctx).Where("repo_id = ? AND commit_sha = ?", repoID, commitSHA).Asc("created_unix").Find(&cs)
	if err != nil {
		return nil, fmt.Errorf("List commit comments: %w", err)
	}
	return cs, nil
}

// ListCommitCommentsByLine returns commit comments for a given repo/commit/path/line
func ListCommitCommentsByLine(ctx context.Context, repoID int64, commitSHA, path string, line int64) ([]*CommitComment, error) {
	cs, err := ListCommitComments(ctx, repoID, commitSHA)
	if err != nil {
		return nil, err
	}
	var res []*CommitComment
	for _, c := range cs {
		if c.Path == path && c.Line == line {
			res = append(res, c)
		}
	}
	return res, nil
}

// UpdateCommitComment updates an existing commit comment
func UpdateCommitComment(ctx context.Context, c *CommitComment) error {
	if c == nil || c.ID == 0 {
		return fmt.Errorf("invalid commit comment")
	}
	if _, err := db.GetEngine(ctx).ID(c.ID).Cols("content", "path", "line", "updated_unix").Update(c); err != nil {
		return fmt.Errorf("Update CommitComment: %w", err)
	}
	return nil
}

// DeleteCommitComment deletes a commit comment by id
func DeleteCommitComment(ctx context.Context, id int64) error {
	c := &CommitComment{ID: id}
	_, err := db.GetEngine(ctx).ID(id).Delete(c)
	if err != nil {
		return fmt.Errorf("Delete CommitComment: %w", err)
	}
	return nil
}

// LoadPoster loads the poster user for the comment
func (c *CommitComment) LoadPoster(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("nil commit comment")
	}
	if c.Poster != nil {
		return nil
	}
	u, err := user_model.GetPossibleUserByID(ctx, c.PosterID)
	if err != nil {
		if user_model.IsErrUserNotExist(err) {
			c.PosterID = user_model.GhostUserID
			c.Poster = user_model.NewGhostUser()
			return nil
		}
		log.Error("getUserByID[%d]: %v", c.PosterID, err)
		return err
	}
	c.Poster = u
	return nil
}

// HashTag returns an id that can be used as a DOM anchor similar to issue comments
func (c *CommitComment) HashTag() string {
	return fmt.Sprintf("commitcomment-%d", c.ID)
}

// UnsignedLine returns absolute line index for use in templates
func (c *CommitComment) UnsignedLine() int64 {
	if c.Line < 0 {
		return -c.Line
	}
	return c.Line
}

// DiffSide returns which side the comment belongs to
func (c *CommitComment) DiffSide() string {
	if c.Line < 0 {
		return "left"
	}
	return "right"
}

// TreePath exposes Path in a field-name compatible way with issue comment templates
func (c *CommitComment) TreePath() string {
	return c.Path
}
