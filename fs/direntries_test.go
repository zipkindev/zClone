package fs_test

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"zclone/fs"
	"zclone/fstest/mockdir"
	"zclone/fstest/mockobject"
)

func TestDirEntriesSort(t *testing.T) {
	a := mockobject.New("a")
	aDir := mockdir.New("a")
	b := mockobject.New("b")
	bDir := mockdir.New("b")
	c := mockobject.New("c")
	cDir := mockdir.New("c")
	anotherc := mockobject.New("c")
	dirEntries := fs.DirEntries{bDir, b, aDir, a, c, cDir, anotherc}

	sort.Stable(dirEntries)

	assert.Equal(t, fs.DirEntries{aDir, a, bDir, b, cDir, c, anotherc}, dirEntries)
}
