package deploy

// PickPort exposes pickPort for whitebox unit testing in deploy_test package.
func PickPort(base int) (int, error) {
	return pickPort(base)
}
