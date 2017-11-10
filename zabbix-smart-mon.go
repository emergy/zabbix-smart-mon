package main

import (
    "fmt"
    "os/exec"
    "log"
    //"github.com/davecgh/go-spew/spew"
    "strings"
    "os"
    "encoding/json"
    "github.com/jessevdk/go-flags"
    "bufio"
    "regexp"
    "io"
    "net"
)

type discoveryType struct {
    Device   string `json:"{#DEV}"`
    AttrId   string `json:"{#ID}"`
    AttrName string `json:"{#NAME}"`
    Worst    string `json:"{#WORST}"`
    Thresh   string `json:"{#THRESH}"`
}

type senderOutput struct {
    Data []discoveryType `json:"data"`
}

type Options struct {
    ZabbixAgentConfig string `short:"c" long:"zabbix-agent-config" description:"Path to zabbix_agentd.conf" default:"/etc/zabbix/zabbix_agentd.conf"`
    HostName          string `short:"H" long:"hostname"            description:"Hostname in Zabbix"         default:"from zabbix_agentd.conf"`
    ZabbixServer    []string `short:"z" long:"zabbix-server"       description:"Zabbix server host"         default:"from zabbix_agentd.conf"`
    DriveList       []string `short:"d" long:"dev"                 description:"Device"                     default:"all"`
    Logfile           string `short:"l" long:"log-file"            description:"Logfile"                    default:"stdout"`
    Verbose           bool   `short:"v" long:"verbose"             description:"Verbose mode"`
}

var zaConfig map[string]string
var opts Options

func init() {
    //_ = spew.Sdump("")

    if _, err := flags.Parse(&opts); err != nil {
        if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
            os.Exit(0)
        } else {
            os.Exit(1)
        }
    }

    zaConfig = readZabbixConfig(opts.ZabbixAgentConfig)

    if opts.ZabbixServer[0] == "from zabbix_agentd.conf" {
        opts.ZabbixServer = strings.Split(zaConfig["Server"], ",")
    }

    if opts.HostName == "from zabbix_agentd.conf" {
        opts.HostName = zaConfig["Hostname"]
    }

}

func main() {
    if len(opts.Logfile) != 0 && opts.Logfile != "stdout" {
        f, err := os.OpenFile(opts.Logfile, os.O_RDWR | os.O_CREATE | os.O_APPEND, 0644)
        if err != nil {
            log.Fatalf("error: %v", err)
        }
        defer f.Close()

        log.SetOutput(f)
    }

    smart := getSMARTinfo()

    var discoveryHeap []discoveryType
    var senderHeap []string

    for _, i := range smart {
        discoveryHeap = append(discoveryHeap, discoveryType{
            Device: i["DEV"],
            AttrId: i["ID"],
            AttrName: i["ATTRIBUTE_NAME"],
            Worst: i["WORST"],
            Thresh: i["THRESH"],
        })

        senderHeap = append(senderHeap, fmt.Sprintf(`"%s" smart.mon.attr.value[%s:%s] "%s"`,
                                                    opts.HostName, i["DEV"], i["ID"], i["VALUE"]))
        senderHeap = append(senderHeap, fmt.Sprintf(`"%s" smart.mon.attr.raw[%s:%s] "%s"`,
                                                    opts.HostName, i["DEV"], i["ID"], i["RAW_VALUE"]))
    }

    discoveryHeap = append(discoveryHeap, discoveryType{
        Device: "dev",
        AttrId: "attrID",
        AttrName: "attrName",
        Worst: "worst",
        Thresh: "thresh",
    })

    discovery, err := json.Marshal(senderOutput{
                                       Data: discoveryHeap,
                                   })
    if err != nil {
        log.Fatal(fmt.Sprintf("Can't build discovery JSON: %s\n", err))
    }

    discoveryString := fmt.Sprintf(`"%s" smart.mon.discovery "%s"`,
                                    zaConfig["Hostname"],
                                    strings.Replace(string(discovery), "\"", "\\\"", -1))

    zabbixSend(opts.ZabbixServer, []string{discoveryString})

    senderHeap = append(senderHeap, fmt.Sprintf(`"%s" smart.mon.attr.value[%s:%s] "%s"`,
                                                opts.HostName, "dev", "attrID", "0"))

    partSize := 200

    for i := len(senderHeap); i >= 0; i -= partSize {
        n := i - partSize
        if (n < 0) {
            n = 0
        }

        zabbixSend(opts.ZabbixServer, senderHeap[n:i])

        if (n == 0) {
            break
        }
    }
}

func zabbixSend(server []string, data []string) {
    for _, zabbixServer := range server {
        conn, err := net.Dial("tcp", zabbixServer + ":10051")
        if err != nil {
            log.Printf("Zabbix server %s not avalible\n", zabbixServer)
            continue
        } else {
            defer conn.Close()
            log.Printf("Sendindg data to %s\n", zabbixServer)
        }

        verbose := ""
        if opts.Verbose == true {
            verbose = "-vv"
        }

        cmd := exec.Command("/usr/bin/zabbix_sender", verbose, "-z", zabbixServer, "-i", "-")
        stdin, err := cmd.StdinPipe()

        if err != nil {
            log.Fatal(err)
        }

        go func() {
            defer stdin.Close()
            if opts.Verbose == true {
                log.Printf("Send data to zabbix server:\n%s", strings.Join(data, "\n"))
            }
            io.WriteString(stdin, strings.Join(data, "\n"))
        }()

        out, err := cmd.CombinedOutput()

        if err != nil {
            log.Printf("Error: %s", err)
        }

        log.Printf("%s", out)
    }
}

func getSMARTinfo() []map[string]string {
    var list []map[string]string

    devList := getDisksList()

    for _, dev := range devList {
        out := execute("/usr/sbin/smartctl -A " + dev)

        readFlag := false
        var names []string

        for _, line := range strings.Split(out, "\n") {
            if len(line) > 0 {
                fields := strings.Fields(line)

                if len(fields) == 10 {
                    attr := make(map[string]string)

                    if readFlag {
                        for i, v := range fields {
                            attr[names[i]] = v
                        }

                        attr["DEV"] = strings.Replace(dev, "/dev/", "", 1)
                        list = append(list, attr)
                    }

                    if fields[0] == "ID#" {
                        readFlag = true
                        names = fields
                        names[0] = "ID"
                    }
                }
            }
        }
    }

    return list
}

func getDisksList() []string {
    out := execute("/usr/sbin/smartctl --scan")

    var rv []string

    for _, i := range strings.Split(out, "\n") {
        if len(i) > 0 {
            dev := strings.Fields(i)[0]

            if len(opts.DriveList) > 0 && opts.DriveList[0] != "all" {
                if ! stringExistInSlice(dev, opts.DriveList) {
                    continue;
                }
            }

            rv = append(rv, dev)
        }
    }

    return rv
}

func stringExistInSlice(val string, slice []string) bool {
    for _, i := range slice {
        if val == i {
            return true
        }
    }

    return false
}

func execute(cmdString string) string {
    parts := strings.Fields(cmdString)

    head := parts[0]
    parts = parts[1:len(parts)]

    out, err := exec.Command(head, parts...).Output()

    if err != nil {
        log.Fatal(err)
    }

    return string(out)
}

func readZabbixConfig(file string) map[string]string {
    rv := make(map[string]string)

    fileh, err := os.Open(file)
    if err != nil {
        log.Fatal(err)
    }

    defer fileh.Close()
    scanner := bufio.NewScanner(fileh)

    for scanner.Scan() {
        line := scanner.Text()

        re := regexp.MustCompile("^\\s*(#|$)")
        next := re.MatchString(line)

        if next {
            continue
        }

        sline := strings.Split(line, "=")
        rv[sline[0]] = sline[1]
    }

    if err := scanner.Err(); err != nil {
        log.Fatal(err)
    }

    return rv
}
