package goxr

import (
	"bytes"
	"errors"
	"github.com/echocat/goxr/common"
	"os"
)

type CombinedBox []Box

func (instance CombinedBox) Open(name string) (common.File, error) {
	for _, box := range instance {
		if f, err := box.Open(name); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return nil, err
		} else {
			return f, nil
		}
	}
	return nil, common.NewPathError("open", name, os.ErrNotExist)
}

func (instance CombinedBox) Info(name string) (common.FileInfo, error) {
	for _, box := range instance {
		if fi, err := box.Info(name); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return nil, err
		} else {
			return fi, nil
		}
	}
	return nil, common.NewPathError("info", name, os.ErrNotExist)
}

func (instance CombinedBox) Close() error {
	var errs []error
	for _, box := range instance {
		if err := box.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) <= 0 {
		return nil
	} else if len(errs) == 1 {
		return errs[0]
	} else {
		buf := new(bytes.Buffer)
		for i, err := range errs {
			if i > 0 {
				common.MustWritef(buf, "\nAND ")
			}
			common.MustWritef(buf, "%v", err)
		}
		return errors.New(buf.String())
	}
}

func (instance CombinedBox) ForEach(predicate common.FilePredicate, callback func(common.FileInfo) error) error {
	for _, box := range instance {
		if ib, ok := box.(Iterable); ok {
			if err := ib.ForEach(predicate, callback); err != nil {
				return err
			}
		} else {
			return ErrBoxIterationNotSupported
		}
	}
	return nil
}

func (instance CombinedBox) With(box Box) CombinedBox {
	return append(instance, box)
}
