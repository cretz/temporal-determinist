package determinist

import (
	"sort"
	"strings"
)

var DefaultIdentRefs = IdentRefs{
	"time.Now":   true,
	"time.Sleep": true,
}

type IdentRefs map[string]bool

func (i IdentRefs) Clone() IdentRefs {
	ret := make(IdentRefs, len(i))
	for k, v := range i {
		ret[k] = v
	}
	return ret
}

func (i IdentRefs) String() string {
	strs := make([]string, 0, len(i))
	for k := range i {
		strs = append(strs, k)
	}
	sort.Strings(strs)
	return strings.Join(strs, ",")
}

func (i IdentRefs) Set(flag string) error {
	i.SetAll(strings.Split(flag, ","))
	return nil
}

func (i IdentRefs) SetAll(refs []string) {
	for _, ref := range refs {
		if strings.HasSuffix(ref, "=false") {
			i[strings.TrimSuffix(ref, "=false")] = false
		} else {
			i[strings.TrimSuffix(ref, "=true")] = true
		}
	}
}
