package client

type Client interface {
	CreateFileSystem(parameters interface{}) (string, error)
	CreateMount(parameters interface{}) (string, error)
}

type Config struct{
	AtAddress string
	ConfigPath string
}
type CephClient struct {
	ceph_mds string

}

func NewCephClient(mds string) *CephClient {
	return &CephClient{
		ceph_mds: mds,
	}
}

func (c *CephClient) CreateFileSystem(parameters interface{})(string, error){
// ceph osd pool create cephfs_data <pg_num>
// ceph osd pool create cephfs_metadata <pg_num>
// ceph fs new cephfs cephfs_metadata cephfs_data
// ceph fs ls
return "", nil
}

func (c *CephClient) CreateMount(parameters interface{})(string, error){
	//sudo mkdir -p /etc/ceph
	//sudo scp {user}@{server-machine}:/etc/ceph/ceph.conf /etc/ceph/ceph.conf
	//sudo scp {user}@{server-machine}:/etc/ceph/ceph.keyring /etc/ceph/ceph.keyring
	//sudo mkdir /home/usernname/cephfs
	//sudo ceph-fuse -m 192.168.0.1:6789 /home/username/cephfs
return "", nil
}
