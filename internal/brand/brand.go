package brand

const (
	ProductName = "Ghost FTP"
	ProductFull = "Ghost FTP file transfer client"
	Company     = ProductName

	Website = "ghostftp.com"
	Support = Website

	WebsiteURL = "https://ghostftp.com/"
	PremiumURL = "https://ghostftp.com/premium/"

	// All desktop update discovery and packages are served from the dedicated
	// first-party update host. The application never sends connection profiles,
	// credentials, file names or transfer metadata to the update service.
	UpdateBaseURL     = "https://update.ghostftp.com/"
	UpdateURL         = UpdateBaseURL
	UpdateManifestURL = "https://update.ghostftp.com/windows/latest.json"
)
