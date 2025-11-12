package common

func ShouldProcessSlash(
	received, slashCmd, testSlashCmd string,
	testingActive bool,
) bool {
	// If testing is active then match to TestSlashCmd
	return (received == testSlashCmd && testingActive) ||
		// If testing is not active then match to SlashCmd
		(received == slashCmd && !testingActive)
}
