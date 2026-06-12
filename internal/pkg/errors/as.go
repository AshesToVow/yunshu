package errors

import stderrors "errors"

func As(err error) (*BizError, bool) {
	var biz *BizError
	ok := stderrors.As(err, &biz)
	return biz, ok
}

func Is(err, target error) bool {
	var biz *BizError
	if stderrors.As(err, &biz) {
		if t, ok := target.(*BizError); ok {
			return biz.Code == t.Code
		}
	}
	return stderrors.Is(err, target)
}
