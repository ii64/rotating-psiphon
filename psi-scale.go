package main
import (
	"os"
	"os/exec"
	"os/signal"
	"fmt"
	"io"
	"flag"
	"bytes"
	"strings"
	"strconv"
	"path/filepath"
	jp "github.com/buger/jsonparser"
)

type HABackend struct {
	Name string
	Port string
	Addr string
}
func (hb HABackend) String() string {
	return `  server ` + hb.Name +"_"+ hb.Port + ` ` + hb.Addr + `:` + hb.Port
}

type HABackends []HABackend
func (hbs HABackends) String() (r string) {
	for _,s := range hbs {
		r += s.String() + "\n"
	}
	return r
}
type HAProxyGen struct {
	cfg         string
	svc_addr    string
	ha_pid_file string
	svc_backend HABackends
}
func NewHAProxyGen() HAProxyGen {
	base := `global
  maxconn 1024 
  daemon
  pidfile $pid_file

defaults
  mode http
  maxconn 1024
  option  httplog
  option  dontlognull
  retries 3
  timeout connect 5s
  timeout client 60s
  timeout server 60s


listen stats
  bind 0.0.0.0:4444
  option http-use-htx
  mode            http
  log             global
  maxconn 10
  timeout client  100s
  timeout server  100s
  timeout connect 100s
  timeout queue   100s
  stats enable
  stats hide-version
  http-request use-service prometheus-exporter if { path /metrics }
  stats refresh 10s
  stats show-node
  stats uri /haproxy?stats


frontend rotating_psiphon
  bind $svc_bind_addr
  default_backend psiphons
  option http_proxy

backend psiphons
  mode http
  acl is_connect_method method CONNECT
  http-request set-uri %[req.hdr(Host)] if is_connect_method
  http-request set-uri http://%[req.hdr(Host)]%[path]%[query] unless is_connect_method
#  http-request set-header X-Dest-Host %[req.hdr(Host)]

  balance leastconn # http://cbonte.github.io/haproxy-dconv/configuration-1.5.html#balance
$svc_backend_server
`
  return HAProxyGen{base, "*:4455", "", HABackends{}}
}
func (hp *HAProxyGen) Generate() string {
	hp.cfg = strings.Replace(hp.cfg, "$svc_bind_addr", hp.svc_addr, 1)
	hp.cfg = strings.Replace(hp.cfg, "$pid_file", hp.ha_pid_file, 1)
	hp.cfg = strings.Replace(hp.cfg, "$svc_backend_server", hp.svc_backend.String(), 1)
	return hp.cfg
}
func (hp *HAProxyGen) SetServiceBindAddr(addr string) {
	hp.svc_addr = addr
}
func (hp *HAProxyGen) SetPidFile(path string) {
	hp.ha_pid_file = path
}
func (hp *HAProxyGen) AddBackend(svr HABackend) {
	hp.svc_backend = append(hp.svc_backend, svr)
}

var rawPCfg = []byte(`{
	"Authorizations":[],
	"ClientPlatform":"Windows_10.0.17763_0",
	"ClientVersion":"154",
	"DataRootDirectory":"$store_dir",
	"DeviceRegion":"",
	"EgressRegion":"SG",
	"EmitDiagnosticNetworkParameters":true,
	"EmitDiagnosticNotices":true,
	"LocalHttpProxyPort": $http_proxy_port,
	"LocalSocksProxyPort": $socks_proxy_port,
	"MigrateDataStoreDirectory": "$store_dir",
	"ObfuscatedServerListDownloadDirectory": "$store_dir/osl",
	"MigrateObfuscatedServerListDownloadDirectory": "$store_dir/osl",
	"RemoteServerListDownloadFilename": "$store_dir/remote_server_list",
	"MigrateRemoteServerListDownloadFilename":"$store_dir/remote_server_list",
	"UpgradeDownloadFilename": "$store_dir/psiphon3.exe.upgrade",
	"MigrateUpgradeDownloadFilename":"$store_dir/psiphon3.exe.upgrade",
	"NetworkID":"949F2E962ED7A9165B81E977A3B4758B",
	"ObfuscatedServerListRootURLs":[{"OnlyAfterAttempts":0,"SkipVerify":false,"URL":"aHR0cHM6Ly9zMy5hbWF6b25hd3MuY29tL3BzaXBob24vd2ViL21qcjQtcDIzci1wdXdsL29zbA=="},{"OnlyAfterAttempts":2,"SkipVerify":true,"URL":"aHR0cHM6Ly93d3cueHlkaWFtb25kZGJleHBlcnQuY29tL3dlYi9tanI0LXAyM3ItcHV3bC9vc2w="},{"OnlyAfterAttempts":2,"SkipVerify":true,"URL":"aHR0cHM6Ly93d3cuZ3BhbGx0aGluZ3NudW1iZXJ3ZWF0aGVyLmNvbS93ZWIvbWpyNC1wMjNyLXB1d2wvb3Ns"},{"OnlyAfterAttempts":2,"SkipVerify":true,"URL":"aHR0cHM6Ly93d3cud2hlZWxyc3NrbWluc2lkZS5jb20vd2ViL21qcjQtcDIzci1wdXdsL29zbA=="}],
	"PropagationChannelId":"92AACC5BABE0944C",
	"RemoteServerListSignaturePublicKey":"MIICIDANBgkqhkiG9w0BAQEFAAOCAg0AMIICCAKCAgEAt7Ls+/39r+T6zNW7GiVpJfzq/xvL9SBH5rIFnk0RXYEYavax3WS6HOD35eTAqn8AniOwiH+DOkvgSKF2caqk/y1dfq47Pdymtwzp9ikpB1C5OfAysXzBiwVJlCdajBKvBZDerV1cMvRzCKvKwRmvDmHgphQQ7WfXIGbRbmmk6opMBh3roE42KcotLFtqp0RRwLtcBRNtCdsrVsjiI1Lqz/lH+T61sGjSjQ3CHMuZYSQJZo/KrvzgQXpkaCTdbObxHqb6/+i1qaVOfEsvjoiyzTxJADvSytVtcTjijhPEV6XskJVHE1Zgl+7rATr/pDQkw6DPCNBS1+Y6fy7GstZALQXwEDN/qhQI9kWkHijT8ns+i1vGg00Mk/6J75arLhqcodWsdeG/M/moWgqQAnlZAGVtJI1OgeF5fsPpXu4kctOfuZlGjVZXQNW34aOzm8r8S0eVZitPlbhcPiR4gT/aSMz/wd8lZlzZYsje/Jr8u/YtlwjjreZrGRmG8KMOzukV3lLmMppXFMvl4bxv6YFEmIuTsOhbLTwFgh7KYNjodLj/LsqRVfwz31PgWQFTEPICV7GCvgVlPRxnofqKSjgTWI4mxDhBpVcATvaoBl1L/6WLbFvBsoAUBItWwctO2xalKxF5szhGm8lccoc5MZr8kfE0uxMgsxz4er68iCID+rsCAQM=",
	"RemoteServerListURLs": [
		{"URL": "aHR0cHM6Ly9zMy5hbWF6b25hd3MuY29tL3BzaXBob24vd2ViL213NHotYTJreC0wd2J6L3NlcnZlcl9saXN0X2NvbXByZXNzZWQ=", "OnlyAfterAttempts": 0, "SkipVerify": false},
		{"URL": "aHR0cHM6Ly93d3cuY29ycG9yYXRlaGlyZXByZXNzdGguY29tL3dlYi9tdzR6LWEya3gtMHdiei9zZXJ2ZXJfbGlzdF9jb21wcmVzc2Vk", "OnlyAfterAttempts": 2, "SkipVerify": true},
		{"URL": "aHR0cHM6Ly93d3cuc3RvcmFnZWpzc3RyYXRlZ2llc2ZhYnVsb3VzLmNvbS93ZWIvbXc0ei1hMmt4LTB3Ynovc2VydmVyX2xpc3RfY29tcHJlc3NlZA==", "OnlyAfterAttempts": 2, "SkipVerify": true},
		{"URL": "aHR0cHM6Ly93d3cuYnJhbmRpbmd1c2FnYW1lcmVwLmNvbS93ZWIvbXc0ei1hMmt4LTB3Ynovc2VydmVyX2xpc3RfY29tcHJlc3NlZA==", "OnlyAfterAttempts": 2, "SkipVerify": true}
	], 
	"SponsorId":"1BC527D3D09985CF",
	"UpgradeDownloadClientVersionHeader":"x-amz-meta-psiphon-client-version",
	"UpgradeDownloadURLs":[
		{"OnlyAfterAttempts":0,"SkipVerify":false,"URL":"aHR0cHM6Ly9zMy5hbWF6b25hd3MuY29tL3BzaXBob24vd2ViL21qcjQtcDIzci1wdXdsL3BzaXBob24zLmV4ZS51cGdyYWRl"},
		{"OnlyAfterAttempts":2,"SkipVerify":true,"URL":"aHR0cHM6Ly93d3cueHlkaWFtb25kZGJleHBlcnQuY29tL3dlYi9tanI0LXAyM3ItcHV3bA=="},
		{"OnlyAfterAttempts":2,"SkipVerify":true,"URL":"aHR0cHM6Ly93d3cuZ3BhbGx0aGluZ3NudW1iZXJ3ZWF0aGVyLmNvbS93ZWIvbWpyNC1wMjNyLXB1d2w="},
		{"OnlyAfterAttempts":2,"SkipVerify":true,"URL":"aHR0cHM6Ly93d3cud2hlZWxyc3NrbWluc2lkZS5jb20vd2ViL21qcjQtcDIzci1wdXds"}
	],
	"UseIndistinguishableTLS":true
}`)

var (
	instanceCount int = 2
	haproxyConfigPath string = "/etc/haproxy/haproxy.cfg"
	holder = make(chan int)
	notifierExit = make(chan int)
)



func CopyFile(src, dst string) (err error) {
	sfi, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !sfi.Mode().IsRegular() {
		return fmt.Errorf("CopyFile: non-regular source file %s (%q)", sfi.Name(), sfi.Mode().String())
	}
	dfi, err := os.Stat(dst)
    if err != nil {
        if !os.IsNotExist(err) {
            return
        }
    } else {
        if !(dfi.Mode().IsRegular()) {
            return fmt.Errorf("CopyFile: non-regular destination file %s (%q)", dfi.Name(), dfi.Mode().String())
        }
        if os.SameFile(sfi, dfi) {
            return
        }
    }
    if err = os.Link(src, dst); err == nil {
        return
    }
    err = copyFileContents(src, dst)
    return
}

func CopyDir(src, dst string) (err error) {
	d, err := os.Open(src)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdir(-1)
	if err != nil {
		return err
	}
	for _, item := range names {
		if item.IsDir() {
			os.MkdirAll(filepath.Join(dst, item.Name()), os.ModeDir)
			CopyDir(filepath.Join(src, item.Name()), filepath.Join(dst, item.Name()))
			continue
		}
		err = copyFileContents(filepath.Join(src, item.Name()), filepath.Join(dst, item.Name()))
		if err != nil {
			continue
		}
	}
	return nil
}

func copyFileContents(src, dst string) (err error) {
    in, err := os.Open(src)
    if err != nil {
        return
    }
    defer in.Close()
    out, err := os.Create(dst)
    if err != nil {
        return
    }
    defer func() {
        cerr := out.Close()
        if err == nil {
            err = cerr
        }
    }()
    if _, err = io.Copy(out, in); err != nil {
        return
    }
    err = out.Sync()
    return
}



func RemoveContents(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			continue
		}
	}
	return nil
}


func main() {
	sigNotification := make(chan os.Signal, 1)
	signal.Notify(sigNotification, os.Interrupt)

	if icount, exist := os.LookupEnv("INSTANCE_COUNT"); exist {
		if rpx, err := strconv.Atoi(icount); err == nil && rpx > 0 {
			instanceCount = rpx
		}
	}
	flag.IntVar(&instanceCount, "count", instanceCount, "Psiphon client count")


	if tmpx, exist := os.LookupEnv("HAPROXY_CONFIG"); exist {
		haproxyConfigPath = tmpx
	}
	flag.StringVar(&haproxyConfigPath, "ha-config", haproxyConfigPath, "HA Proxy configuration path destination")

	flag.Parse()

	pcid, err := jp.GetString(rawPCfg, "PropagationChannelId")
	if err != nil {
		pcid = "NULL"
	}
	spid, err := jp.GetString(rawPCfg, "SponsorId")
	if err != nil {
		spid = "NULL"
	}
	initialLocalHttpProxyPort, err := jp.GetInt(rawPCfg, "LocalHttpProxyPort")
	if err != nil {
		initialLocalHttpProxyPort = 8080
	}
	initialLocalSocksProxyPort, err := jp.GetInt(rawPCfg, "LocalSocksProxyPort")
	if err != nil {
		initialLocalSocksProxyPort = 1080
	}
	fmt.Printf("%s\n", colArrange([][]string{
		{"Instance to deploy:", fmt.Sprintf("%d", instanceCount)},
		{"PropagationChannelId:", pcid},
		{"SponsorId:", spid},
		{"Init LocalHttpProxyPort:", fmt.Sprintf("%d", initialLocalHttpProxyPort)},
		{"Init LocalSocksProxyPort:", fmt.Sprintf("%d", initialLocalSocksProxyPort)},
	}))

	doExit := func() {
		for insId := 0; insId < instanceCount; insId++ {
			//_ = insId
			os.Remove(fmt.Sprintf("./tmp_%d.json", insId))
			os.Remove(fmt.Sprintf("./desktop_%d",insId))
			RemoveContents(fmt.Sprintf("desktop_%d",insId))
		//	os.Exit(2)
		}
	}
	go func() {
		for sig := range sigNotification {
			_ = sig
			doExit()
			holder <- 1
		}
	}()
	go func() {
		<-notifierExit
		doExit()
		holder <- 1
	}()

	fcfg, err := os.Create(haproxyConfigPath)
	if err != nil {
		fmt.Printf("error open haproxy.cfg: %s", err)
		os.Exit(2)
	}

	hacfg := NewHAProxyGen()
	hacfg.SetPidFile("/tmp/haproxy.pid")
	hacfg.SetServiceBindAddr("*:4455")
	for insId := 0; insId < instanceCount; insId++ {
		fmt.Printf("Creating %d... ", insId)
		f, err := os.Create(fmt.Sprintf("tmp_%d.json", insId))
		if err != nil {
			fmt.Printf("[%d] error open file: %s\n", insId, err)
			continue
		}
		dirPath := fmt.Sprintf("./desktop_%d",insId)
		os.MkdirAll(dirPath, os.ModeDir)
		fmt.Printf("copy dir error: %s\n", CopyDir("desktop", dirPath))

		localRawPCfg := bytes.Replace(rawPCfg, []byte("$http_proxy_port"),  []byte(fmt.Sprintf("%d",initialLocalHttpProxyPort)), -1)
		localRawPCfg = bytes.Replace(localRawPCfg, []byte("$socks_proxy_port"), []byte(fmt.Sprintf("%d",initialLocalSocksProxyPort)), -1)
		localRawPCfg = bytes.Replace(localRawPCfg, []byte("$store_dir"), []byte(dirPath), -1)
		f.Write(localRawPCfg)
		f.Close()
		var (
			runResult = make(chan int)
			erro error
		)
		c := exec.Command("psiphon-tunnel-core", "-config", fmt.Sprintf("tmp_%d.json", insId))
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		go func() {
			if err := c.Run(); err != nil {
				erro = err
				fmt.Printf("Failed.\n")
				fmt.Printf(" - err: %s\n", erro)
				runResult <- 2
			}else{
				runResult <- 0
			}
		}()
		// just pass the failed port
		hacfg.AddBackend(HABackend{Name:fmt.Sprintf("backend%d",insId), Addr: "127.0.0.1", Port: fmt.Sprintf("%d", initialLocalHttpProxyPort)})
		initialLocalHttpProxyPort = initialLocalHttpProxyPort + 1
		initialLocalSocksProxyPort = initialLocalSocksProxyPort + 1
		fmt.Printf("Done\n")
	}
	fcfg.Write([]byte(hacfg.Generate()))
	fcfg.Close()
	hasvc := exec.Command("haproxy", "-D", "-f", haproxyConfigPath)
	hasvc.Stderr = os.Stderr
	go func() {
		if err := hasvc.Run(); err != nil {
			fmt.Printf("Failed to run HA Proxy\n")
			fmt.Printf(" - err: %s\n", err)
			//notifierExit <- 1
		}
	}()
	<-holder
}


func colArrange(rows [][]string) (r string) {
	maxSz := make([]int, len(rows[0]))
	for _, row := range rows {
		for i, col := range row {
			if len(col) > maxSz[i] {
				maxSz[i] = len(col)
			}
		}
	}
	for _, row := range rows {
		for i, col := range row {
			missingSpace := strings.Repeat(" ", maxSz[i] - len(col) + 2)
			r += col + missingSpace
		}
		r += "\n"
	}
	return r
}
