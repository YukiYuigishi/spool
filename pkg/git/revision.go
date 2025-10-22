package git

import (
)

const (
	HEAD             = "HEAD"
	FETCH_HEAD       = "FETCH_HEAD"
	ORIG_HEAD        = "ORIG_HEAD"
	MERGE_HEAD       = "MERGE_HEAD"
	REBASE_HEAD      = "REBASE_HEAD"
	REVERT_HEAD      = "REVERT_HEAD"
	CHERRY_PICK_HEAD = "CHERRY_PICK_HEAD"
	BISECT_HEAD      = "BISECT_HEAD"
	AUTO_MERGE       = "AUTO_MERGE"
)

func NewGitObject(body []byte)(error){
	return  nil
}
