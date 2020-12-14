package viewmodels

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/wagoodman/dive/dive/filetree"
	"github.com/wagoodman/dive/dive/image"
	"regexp"
)

type FilterModel interface {
	SetFilter(r *regexp.Regexp)
	GetFilter() *regexp.Regexp
}


type LayersModel interface {
	SetLayerIndex(index int) bool
	GetCompareIndicies() filetree.TreeIndexKey
	GetCurrentLayer() *image.Layer
	GetPrintableLayers() []fmt.Stringer
}


type TreeViewModel struct {
	currentTree *filetree.FileTree
	cache filetree.Comparer
	// Make this an interface that is composed with the FilterView
	FilterModel
	LayersModel
}


func NewTreeViewModel(cache filetree.Comparer,lModel LayersModel, fmodel FilterModel) (*TreeViewModel, error) {
	curTreeIndex := filetree.NewTreeIndexKey(0,0,0,0)
	tree, err := cache.GetTree(curTreeIndex)
	if err != nil {
		return nil, err
	}
	return &TreeViewModel{
		currentTree: tree,
		cache: cache,
		FilterModel: fmodel,
		LayersModel: lModel,
	}, nil
}

func (tvm *TreeViewModel) StringBetween(startRow , stopRow int, showAttributes bool) string {
	return tvm.currentTree.StringBetween(startRow, stopRow, showAttributes)
}

func (tvm *TreeViewModel) VisitDepthParentFirst(visitor filetree.Visitor, evaluator filetree.VisitEvaluator) error {
	return tvm.currentTree.VisitDepthParentFirst(visitor, evaluator)
}
func (tvm *TreeViewModel) VisitDepthChildFirst(visitor filetree.Visitor, evaluator filetree.VisitEvaluator) error {
	return tvm.currentTree.VisitDepthChildFirst(visitor, evaluator)

}
func (tvm *TreeViewModel) RemovePath(path string) error {
	return tvm.currentTree.RemovePath(path)
}
func (tvm *TreeViewModel) VisibleSize() int {
	return tvm.currentTree.VisibleSize()
}

func (tvm *TreeViewModel) SetFilter(filterRegex *regexp.Regexp) {
	tvm.FilterModel.SetFilter(filterRegex)
	if err := tvm.FilterUpdate(); err != nil {
		panic(err)
	}
}

func (tvm *TreeViewModel) FilterUpdate() error {
	// keep the t selection in parity with the current DiffType selection
	filter := tvm.GetFilter()
	err := tvm.currentTree.VisitDepthChildFirst(func(node *filetree.FileNode) error {

		visibleChild := false
		if filter == nil {
			node.Data.ViewInfo.Hidden = false
			return nil
		}

		for _, child := range node.Children {
			if !child.Data.ViewInfo.Hidden {
				visibleChild = true
				node.Data.ViewInfo.Hidden = false
				return nil
			}
		}

		if !visibleChild { // hide nodes that do not match the current file filter regex (also don't unhide nodes that are already hidden)
			match := filter.FindString(node.Path())
			node.Data.ViewInfo.Hidden = len(match) == 0
		}
		return nil
	}, nil)

	if err != nil {
		logrus.Errorf("unable to propagate t model tree: %+v", err)
		return err
	}

	return nil
}

// Override functions

func (tvm *TreeViewModel) SetLayerIndex(index int) bool {
	if tvm.LayersModel.SetLayerIndex(index) {
		err := tvm.setCurrentTree(tvm.GetCompareIndicies())
		if err != nil {
			// TODO handle error here
			return false
		}
		return true
	}
	return false
}

func (tvm *TreeViewModel) setCurrentTree(key filetree.TreeIndexKey) error {
	collapsedList := map[string]interface{}{}

	newTree, err := tvm.cache.GetTree(key)
	if err != nil {
		return err
	}

	evaluateFunc := func(node *filetree.FileNode) bool {
		if node.Parent != nil && node.Parent.Data.ViewInfo.Hidden {
			return false
		}
		return true
	}

	tvm.currentTree.VisitDepthParentFirst(func(node *filetree.FileNode) error {
		if node.Data.ViewInfo.Collapsed {
			collapsedList[node.Path()] = true
		}
		return nil
	},evaluateFunc)

	newTree.VisitDepthParentFirst(func(node *filetree.FileNode) error {
		_, ok := collapsedList[node.Path()]
		if ok {
			node.Data.ViewInfo.Collapsed = true
		}
		return nil
	}, evaluateFunc)

	tvm.currentTree = newTree
	if err := tvm.FilterUpdate(); err != nil {
		return err
	}
	return nil
}