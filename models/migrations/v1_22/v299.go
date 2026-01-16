// Copyright 2026 The Gitea Authors.
// SPDX-License-Identifier: MIT

package v1_22

import (
	"xorm.io/xorm"
)

// commitCommentReaction is the minimal schema used for migration only
type commitCommentReaction struct {
	ID               int64  `xorm:"pk autoincr"`
	Type             string `xorm:"INDEX UNIQUE(s) NOT NULL"`
	CommitCommentID  int64  `xorm:"INDEX UNIQUE(s) NOT NULL"`
	UserID           int64  `xorm:"INDEX UNIQUE(s) NOT NULL"`
	OriginalAuthorID int64  `xorm:"INDEX UNIQUE(s) NOT NULL DEFAULT(0)"`
	OriginalAuthor   string `xorm:"INDEX UNIQUE(s)"`
	CreatedUnix      int64  `xorm:"INDEX created"`
}

// AddCommitCommentReactionTable creates the commit_comment_reaction table if it doesn't exist
func AddCommitCommentReactionTable(x *xorm.Engine) error {
	// Sync will create the table or add missing columns
	if err := x.Sync(new(commitCommentReaction)); err != nil {
		return err
	}
	return nil
}