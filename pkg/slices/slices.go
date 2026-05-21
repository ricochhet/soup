package slices

import "slices"

func Matches(sliceA, sliceB []string) int {
	seen := make(map[string]struct{}, len(sliceB))
	for _, e := range sliceB {
		seen[e] = struct{}{}
	}

	var i int

	for _, part := range sliceA {
		if _, found := seen[part]; found {
			i++
		}
	}

	return i
}

func Replace[T comparable](slice, old, replace []T) []T {
	for i := 0; i <= len(slice)-len(old); i++ {
		if slices.Equal(slice[i:i+len(old)], old) {
			return slices.Concat(slice[:i], append(replace, slice[i+len(old):]...))
		}
	}

	return slice
}

func Subslice[T comparable](slice, contains []T) bool {
	if len(contains) == 0 || len(contains) > len(slice) {
		return false
	}

	for i := 0; i <= len(slice)-len(contains); i++ {
		match := true

		for j := range contains {
			if slice[i+j] != contains[j] {
				match = false
				break
			}
		}

		if match {
			return true
		}
	}

	return false
}

func Move[T comparable](slice []T, element T, index int) []T {
	cur := -1

	for i, s := range slice {
		if s == element {
			cur = i
			break
		}
	}

	if cur == -1 {
		return slice
	}

	slice = slices.Delete(slice, cur, cur+1)
	if index >= len(slice) {
		slice = append(slice, element)
	} else {
		slice = append(slice[:index], append([]T{element}, slice[index:]...)...)
	}

	return slice
}
