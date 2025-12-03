package cloud

const (
	AWS       string = "aws"
	Azure     string = "azure"
	Docker    string = "docker"
	GCP       string = "gcp"
	OpenStack string = "openstack"
	VSphere   string = "vsphere"
	Remote    string = "k0sproject-k0smotron"
	Adopted   string = "internal"
)

func IsValidName(cloud string) bool {
	switch cloud {
	case AWS, Azure, Docker, GCP, OpenStack, VSphere, Remote, Adopted:
		return true
	default:
		return false
	}
}
