package sensitive

import "github.com/importcjj/sensitive"

type FilterImpl struct {
	filter *sensitive.Filter
}

var f *FilterImpl

func InitFilter() {
	filter := sensitive.New()
	// 加载自定义词库
	filter.LoadWordDict("./data/sensitive/word.txt")
	filter.LoadWordDict("./data/sensitive/custom.txt")
	f = &FilterImpl{
		filter: filter,
	}
}

func HaveSensitive(input string) bool {
	ok, _ := f.filter.Validate(input)
	return !ok
}
