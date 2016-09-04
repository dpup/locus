package upstream

// Register an upstream factor that matches http and https URLs.
func init() {
	Register(
		`^https?://[[:alnum:]\.\-]+/?.*$`,
		func(url string, settings map[string]string) (Source, error) {
			return FixedSet(url), nil
		})
}

// Single returns a provider that only has one upstream.
func Single(urlStr string) Provider {
	return First(FixedSet(urlStr))
}
