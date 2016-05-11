package client

type Client interface {
	IsFilesystemMounted() (bool, error)
	MountFileSystem(string) (string, error)
	CreateShare(string) (string, error)
	DeleteShare(string) error
}

type CephClient struct {
	ceph_mds string
}

func NewCephClient(mds string) *CephClient {
	return &CephClient{
		ceph_mds: mds,
	}
}

func (c *CephClient) MountFileSystem(targetLocation string) (string, error) {
	// ceph osd pool create cephfs_data <pg_num>
	// ceph osd pool create cephfs_metadata <pg_num>
	// ceph fs new cephfs cephfs_metadata cephfs_data
	// ceph fs ls
	return "", nil
}

func (c *CephClient) CreateShare(shareName string) (string, error) {
	//sudo mkdir -p /etc/ceph
	//sudo scp {user}@{server-machine}:/etc/ceph/ceph.conf /etc/ceph/ceph.conf
	//sudo scp {user}@{server-machine}:/etc/ceph/ceph.keyring /etc/ceph/ceph.keyring
	//sudo mkdir /home/usernname/cephfs
	//sudo ceph-fuse -m 192.168.0.1:6789 /home/username/cephfs
	return "", nil
}
func (c *CephClient) DeleteShare(shareName string) error {
	//sudo mkdir -p /etc/ceph
	//sudo scp {user}@{server-machine}:/etc/ceph/ceph.conf /etc/ceph/ceph.conf
	//sudo scp {user}@{server-machine}:/etc/ceph/ceph.keyring /etc/ceph/ceph.keyring
	//sudo mkdir /home/usernname/cephfs
	//sudo ceph-fuse -m 192.168.0.1:6789 /home/username/cephfs
	return nil
}
