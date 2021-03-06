package configuration

import (
	"fmt"
	"github.com/echocat/goxr"
	"github.com/urfave/cli"
	"os"
	"regexp"
)

type Paths struct {
	Catchall    Catchall       `yaml:"catchall,omitempty"`
	Index       *string        `yaml:"index,omitempty"`
	StatusCodes map[int]string `yaml:"statusCodes,omitempty"`
	Includes    *[]string      `yaml:"includes,omitempty"`
	Excludes    *[]string      `yaml:"excludes,omitempty"`

	defaultFallback     string
	includesRegexpCache *[]*regexp.Regexp
	excludesRegexpCache *[]*regexp.Regexp
}

func (instance Paths) GetIndex() string {
	r := instance.Index
	if r == nil {
		return instance.defaultFallback
	}
	return *r
}

func (instance Paths) GetStatusCodes() map[int]string {
	r := instance.StatusCodes
	if r == nil {
		return make(map[int]string)
	}
	return r
}

func (instance Paths) cloneStatusCodes() map[int]string {
	r := map[int]string{}
	for k, v := range instance.GetStatusCodes() {
		r[k] = v
	}
	return r
}

func (instance Paths) GetIncludes() []string {
	r := instance.Includes
	if r == nil {
		return []string{}
	}
	return *r
}

func (instance Paths) cloneIncludes() []string {
	i := instance.GetIncludes()
	r := make([]string, len(i))
	for i, v := range i {
		r[i] = v
	}
	return r
}

func (instance Paths) GetExcludes() []string {
	r := instance.Excludes
	if r == nil {
		return []string{
			regexp.QuoteMeta("/" + LocationInBox),
		}
	}
	return *r
}

func (instance Paths) cloneExcludes() []string {
	i := instance.GetExcludes()
	r := make([]string, len(i))
	for i, v := range i {
		r[i] = v
	}
	return r
}

func (instance *Paths) FindStatusCode(code int) string {
	r := instance.StatusCodes
	if r == nil {
		return ""
	}
	return r[code]
}

func (instance *Paths) PathAllowed(candidate string) (bool, error) {
	includes := instance.includesRegexpCache
	excludes := instance.excludesRegexpCache
	for i := 0; i < 100 && includes == nil; i++ {
		if errs := instance.rebuildIncludesCache(); len(errs) > 0 {
			return false, errs[0]
		}
		includes = instance.includesRegexpCache
	}
	for i := 0; i < 100 && excludes == nil; i++ {
		if errs := instance.rebuildExcludesCache(); len(errs) > 0 {
			return false, errs[0]
		}
		excludes = instance.excludesRegexpCache
	}

	if includes != nil && len(*includes) > 0 {
		foundMatch := false
		for _, r := range *includes {
			if r.MatchString(candidate) {
				foundMatch = true
			}
		}
		if !foundMatch {
			return false, nil
		}
	}

	if excludes != nil && len(*excludes) > 0 {
		foundMatch := false
		for _, r := range *excludes {
			if r.MatchString(candidate) {
				foundMatch = true
			}
		}
		if foundMatch {
			return false, nil
		}
	}

	return true, nil
}

func (instance *Paths) Validate(using goxr.Box) (errors []error) {
	errors = append(errors, instance.Catchall.Validate(using)...)
	errors = append(errors, instance.validateIndex(using)...)
	errors = append(errors, instance.validateStatusCodes(using)...)
	errors = append(errors, instance.rebuildIncludesCache()...)
	errors = append(errors, instance.rebuildExcludesCache()...)
	return
}

func (instance *Paths) validateIndex(using goxr.Box) (errors []error) {
	r := instance.Index
	if r != nil {
		if _, err := using.Info(*r); os.IsNotExist(err) {
			errors = append(errors, fmt.Errorf(`paths.index = "%s" - path does not exist in box`, *r))
		} else if err != nil {
			errors = append(errors, fmt.Errorf(`paths.index = "%s" - cannot read path information: %v`, *r, err))
		}
	} else {
		if _, err := using.Info("/index.html"); os.IsNotExist(err) {
			instance.defaultFallback = ""
		} else if err != nil {
			errors = append(errors, fmt.Errorf(`cannot read path information for default index "/index.html": %v`, err))
		} else {
			instance.defaultFallback = "/index.html"
		}
	}
	return
}

func (instance *Paths) validateStatusCodes(using goxr.Box) (errors []error) {
	r := instance.GetStatusCodes()
	for code, path := range r {
		if _, err := using.Info(path); os.IsNotExist(err) {
			errors = append(errors, fmt.Errorf(`paths.statusCodes[%d]= "%s" - path does not exist in box`, code, path))
		} else if err != nil {
			errors = append(errors, fmt.Errorf(`paths.statusCodes[%d]= "%s" - cannot read path information: %v`, code, path, err))
		}
	}
	return
}

func (instance *Paths) rebuildIncludesCache() (errors []error) {
	r := instance.GetIncludes()
	rs := make([]*regexp.Regexp, len(r))
	for i, pattern := range r {
		if crx, err := regexp.Compile(pattern); err != nil {
			errors = append(errors, fmt.Errorf(`paths.includes[%d]= "%s" - pattern invalid: %v`, i, pattern, err))
		} else {
			rs[i] = crx
		}
	}
	instance.includesRegexpCache = &rs
	return
}

func (instance *Paths) rebuildExcludesCache() (errors []error) {
	r := instance.GetExcludes()
	rs := make([]*regexp.Regexp, len(r))
	for i, pattern := range r {
		if crx, err := regexp.Compile(pattern); err != nil {
			errors = append(errors, fmt.Errorf(`paths.excludes[%d]= "%s" - pattern invalid: %v`, i, pattern, err))
		} else {
			rs[i] = crx
		}
	}
	instance.excludesRegexpCache = &rs
	return
}

func (instance Paths) Merge(with Paths) Paths {
	result := instance

	result.Catchall = result.Catchall.Merge(with.Catchall)

	if with.Index != nil {
		result.Index = &(*with.Index)
		result.defaultFallback = ""
	}
	if with.StatusCodes != nil {
		result.StatusCodes = with.cloneStatusCodes()
	}
	if with.Includes != nil {
		v := with.cloneIncludes()
		result.Includes = &v
		result.includesRegexpCache = nil
	}
	if with.Excludes != nil {
		v := with.cloneExcludes()
		result.Excludes = &v
		result.excludesRegexpCache = nil
	}

	return result
}

func (instance *Paths) Flags() []cli.Flag {
	return []cli.Flag{}
}
