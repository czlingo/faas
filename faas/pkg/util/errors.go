package util

import "k8s.io/apimachinery/pkg/api/errors"

func IgnoreNotFound(err error) error {
	if err == nil {
		return nil
	}

	if errors.IsNotFound(err) {
		return nil
	}
	return err
}
