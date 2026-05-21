package maps

import (
	"encoding/json"
	"maps"
	"reflect"
	"strings"
)

func Kvp(kvps []string) (map[string]string, error) {
	m := map[string]string{}

	for _, kvp := range kvps {
		kvp := strings.SplitN(kvp, "=", 2)
		if len(kvp) != 2 {
			continue
		}

		m[kvp[0]] = kvp[1]
	}

	return m, nil
}

func Slice(kvps map[string]string) []string {
	result := make([]string, 0, len(kvps))
	for _, kvp := range kvps {
		result = append(result, kvp)
	}

	return result
}

func Map(data []byte) (map[string]any, error) {
	var b map[string]any
	if err := json.Unmarshal(data, &b); err != nil {
		return nil, err
	}

	return b, nil
}

func Merge[K comparable, V any](mapA, mapB map[K]V) map[K]V {
	m := make(map[K]V, len(mapA)+len(mapB))

	maps.Copy(m, mapA)
	maps.Copy(m, mapB)

	return m
}

func AppendNewByKey[T any, K comparable](mapA, mapB []T, key func(T) K) []T {
	seen := make(map[K]struct{}, len(mapA)+len(mapB))
	result := []T{}

	for _, v := range mapA {
		k := key(v)
		seen[k] = struct{}{}

		result = append(result, v)
	}

	for _, v := range mapB {
		k := key(v)
		if _, exists := seen[k]; !exists {
			result = append(result, v)
			seen[k] = struct{}{}
		}
	}

	return result
}

func AppendOverwriteByKey[T any, K comparable](mapA, mapB []T, key func(T) K) []T {
	seen := make(map[K]T, len(mapA)+len(mapB))
	order := []K{}

	for _, v := range mapA {
		k := key(v)
		seen[k] = v
		order = append(order, k)
	}

	for _, v := range mapB {
		k := key(v)
		if _, ok := seen[k]; !ok {
			order = append(order, k)
		}

		seen[k] = v
	}

	result := make([]T, 0, len(seen))
	for _, k := range order {
		result = append(result, seen[k])
	}

	return result
}

func IsZero(v any) bool { return reflect.ValueOf(v).IsZero() }
