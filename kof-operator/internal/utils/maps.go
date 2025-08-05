package utils

func InitMapValue[K comparable, V any](m map[K]V, key K, newV func() V) {
	if _, ok := m[key]; !ok {
		m[key] = newV()
	}
}
