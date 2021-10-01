package determinism

import (
	"sort"
	"strings"
)

var DefaultIdentRefs = IdentRefs{
	"os.Stderr":  true,
	"os.Stdin":   true,
	"os.Stdout":  true,
	"time.Now":   true,
	"time.Sleep": true,
	// We mark these as deterministic since they give so many false positives
	// "(*fmt.pp).printValue": false,
	"(reflect.Value).Interface": false,
	"runtime.Caller":            false,
	// We are considering the global pseudorandom as non-deterministic by default
	// since it's global (even if they set a seed), but we allow use of a manually
	// instantiated random instance that may have a localized, fixed seed
	"math/rand.globalRand": true,
	// Even though the global crypto rand reader var can be replaced, it's good
	// to disallow it by default
	"crypto/rand.Reader": true,
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
