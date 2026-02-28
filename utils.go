package main

func append2[S ~[]E, E any](s *S, e ...E) {
	*s = append(*s, e...)
}

