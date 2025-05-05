package cmd

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/optiflow-os/tracelens-cli/pkg/api"
	"github.com/optiflow-os/tracelens-cli/pkg/exitcode"
	"github.com/optiflow-os/tracelens-cli/pkg/offline"

	viperini "github.com/go-viper/encoding/ini"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	iniv1 "gopkg.in/ini.v1"
)

// defaultConfigSection 是 tracelens ini 配置文件中的默认部分。
const defaultConfigSection = "settings"

// NewRootCMD 创建 rootCmd，当不带任何子命令调用时，它代表基本命令。
func NewRootCMD() *cobra.Command {
	multilineOption := iniv1.LoadOptions{AllowPythonMultilineValues: true}
	iniCodec := viperini.Codec{LoadOptions: multilineOption}

	codecRegistry := viper.NewCodecRegistry()
	if err := codecRegistry.RegisterCodec("ini", iniCodec); err != nil {
		log.Fatalf("注册 ini 编解码器失败: %s", err)
	}

	v := viper.NewWithOptions(viper.WithCodecRegistry(codecRegistry))

	cmd := &cobra.Command{
		Use:   "tracelens-cli",
		Short: "Command line interface used by all tl text editor plugins.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := RunE(cmd, v); err != nil {
				var errexitcode exitcode.Err

				if errors.As(err, &errexitcode) {
					os.Exit(errexitcode.Code)
				}

				os.Exit(exitcode.ErrGeneric)
			}

			os.Exit(exitcode.Success)

			return nil
		},
	}

	setFlags(cmd, v)

	return cmd
}

// setFlags 设置命令行标志。
func setFlags(cmd *cobra.Command, v *viper.Viper) {
	flags := cmd.Flags()
	// 可选的备用分支名称。自动检测的分支优先。
	flags.String("alternate-branch", "", "Optional alternate branch name. Auto-detected branch takes priority.")
	// 可选的备用语言名称。自动检测的语言优先。
	flags.String("alternate-language", "", "Optional alternate language name. Auto-detected language takes priority.")
	// 可选的备用项目名称。自动检测的项目优先。
	flags.String("alternate-project", "", "Optional alternate project name. Auto-detected project takes priority.")
	// 发送心跳和获取代码统计时使用的API基本URL。默认为 https://api.wakatime.com/api/v1/。
	flags.String(
		"api-url",
		"",
		"API base url used when sending heartbeats and fetching code stats. Defaults to https://api.wakatime.com/api/v1/.",
	)
	// （已弃用）发送心跳和获取代码统计时使用的API基本URL。默认为 https://api.wakatime.com/api/v1/。
	flags.String(
		"apiurl",
		"",
		"(deprecated) API base url used when sending heartbeats and fetching code stats. Defaults to"+
			" https://api.wakatime.com/api/v1/.",
	)
	// 此心跳活动的类别。可以是"coding"、"building"、"indexing"、"debugging"、"learning"、
	// "meeting"、"planning"、"researching"、"communicating"、"supporting"、
	// "advising"、"running tests"、"writing tests"、"manual testing"、
	// "writing docs"、"code reviewing"、"browsing"、"translating"或"designing"。默认为"coding"。
	flags.String(
		"category",
		"",
		"Category of this heartbeat activity. Can be \"coding\","+
			" \"building\", \"indexing\", \"debugging\", \"learning\","+
			" \"meeting\", \"planning\", \"researching\", \"communicating\", \"supporting\" "+
			" \"advising\", \"running tests\", \"writing tests\", \"manual testing\","+
			" \"writing docs\", \"code reviewing\", \"browsing\","+
			" \"translating\", or \"designing\". Defaults to \"coding\".",
	)
	// 可选的配置文件。默认为'~/.wakatime.cfg'。
	flags.String("config", "", "Optional config file. Defaults to '~/.wakatime.cfg'.")
	// 可选的内部配置文件。默认为'~/.wakatime/wakatime-internal.cfg'。
	flags.String("internal-config", "", "Optional internal config file. Defaults to '~/.wakatime/wakatime-internal.cfg'.")
	// 打印给定配置键的值，然后退出。
	flags.String("config-read", "", "Prints value for the given config key, then exits.")
	// 读取或写入配置键时的可选配置部分。默认为[settings]。
	flags.String(
		"config-section",
		defaultConfigSection,
		"Optional config section when reading or writing a config key. Defaults to [settings].",
	)
	// 将值写入配置键，然后退出。需要两个参数，键和值。
	flags.StringToString(
		"config-write",
		nil,
		"Writes value to a config key, then exits. Expects two arguments, key and value.",
	)
	// 当前文件中的可选光标位置。
	flags.Int("cursorpos", 0, "Optional cursor position in the current file.")
	// 禁用离线时间记录，而不是排队记录的时间。
	flags.Bool("disable-offline", false, "Disables offline time logging instead of queuing logged time.")
	// （已弃用）禁用离线时间记录，而不是排队记录的时间。
	flags.Bool("disableoffline", false, "(deprecated) Disables offline time logging instead of queuing logged time.")
	// 心跳的文件的绝对路径。当--entity-type不是文件时，也可以是url、域或应用程序。
	flags.String(
		"entity",
		"",
		"Absolute path to file for the heartbeat. Can also be a url, domain or app when --entity-type is not file.",
	)
	// 此心跳的实体类型。可以是"file"、"domain"、"url"或"app"。默认为"file"。
	flags.String(
		"entity-type",
		"",
		"Entity type for this heartbeat. Can be \"file\", \"domain\", \"url\", or \"app\". Defaults to \"file\".",
	)
	// 要排除记录的文件名模式。POSIX正则表达式语法。可以多次使用。
	flags.StringSlice(
		"exclude",
		nil,
		"Filename patterns to exclude from logging. POSIX regex syntax."+
			" Can be used more than once.",
	)
	// 设置时，任何无法检测到项目的活动都将被忽略。
	flags.Bool(
		"exclude-unknown-project",
		false,
		"When set, any activity where the project cannot be detected will be ignored.",
	)
	// 从STDIN读取额外的心跳作为JSON数组，直到EOF。
	flags.Bool("extra-heartbeats", false, "Reads extra heartbeats from STDIN as a JSON array until EOF.")
	// （已弃用）心跳的文件的绝对路径。当--entity-type不是文件时，也可以是url、域或应用程序。
	flags.String(
		"file",
		"",
		"(deprecated) Absolute path to file for the heartbeat."+
			" Can also be a url, domain or app when --entity-type is not file.")
	// 打印团队中给定实体的顶级开发人员，然后退出。
	flags.Bool("file-experts", false, "Prints the top developer within a team for the given entity, then exits.")
	// 启用从文件内容检测语言。
	flags.Bool(
		"guess-language",
		false,
		"Enable detecting language from file contents.")
	// 仅每这些秒将心跳同步到API一次，而不是保存到离线数据库。默认为60秒。使用零禁用。
	flags.Int(
		"heartbeat-rate-limit-seconds",
		offline.RateLimitDefaultSeconds,
		fmt.Sprintf("Only sync heartbeats to the API once per these seconds, instead"+
			" saving to the offline db. Defaults to %d. Use zero to disable.",
			offline.RateLimitDefaultSeconds),
	)
	// 混淆分支名称。不会将修订控制分支名称发送到api。
	flags.String("hide-branch-names", "", "Obfuscate branch names. Will not send revision control branch names to api.")
	// 混淆文件名。不会将文件名发送到api。
	flags.String("hide-file-names", "", "Obfuscate filenames. Will not send file names to api.")
	// （已弃用）混淆文件名。不会将文件名发送到api。
	flags.String("hide-filenames", "", "(deprecated) Obfuscate filenames. Will not send file names to api.")
	flags.String("hidefilenames", "", "(deprecated) Obfuscate filenames. Will not send file names to api.")
	// 设置时，发送文件相对于项目文件夹的路径。例如：/User/me/projects/bar/src/file.ts作为src/file.ts发送，因此服务器永远不会看到完整路径。
	// 当无法检测到项目文件夹时，仅发送文件名。例如：file.ts。
	flags.Bool(
		"hide-project-folder",
		false,
		"When set, send the file's path relative to the project folder."+
			" For ex: /User/me/projects/bar/src/file.ts is sent as src/file.ts so the server never sees the full path."+
			" When the project folder cannot be detected, only the file name is sent. For ex: file.ts.")
	// 混淆项目名称。当检测到项目文件夹时，不使用文件夹名称作为项目，而是创建一个.wakatime-project文件，其中包含随机项目名称。
	flags.String(
		"hide-project-names",
		"",
		"Obfuscate project names. When a project folder is detected instead of"+
			" using the folder name as the project, a .wakatime-project file is"+
			" created with a random project name.",
	)
	// 本地计算机的可选名称。默认为从系统读取的本地计算机名称。
	flags.String("hostname", "", "Optional name of local machine. Defaults to local machine name read from system.")
	// 要记录的文件名模式。当与--exclude一起使用时，匹配include的文件仍将被记录。POSIX正则表达式语法。可以多次使用。
	flags.StringSlice(
		"include",
		nil,
		"Filename patterns to log. When used in combination with"+
			" --exclude, files matching include will still be logged."+
			" POSIX regex syntax. Can be used more than once.",
	)
	// 禁用跟踪文件夹，除非它们包含.wakatime-project文件。默认为false。
	flags.Bool(
		"include-only-with-project-file",
		false,
		"Disables tracking folders unless they contain a .wakatime-project file. Defaults to false.",
	)
	// 通常，不存在于磁盘上的文件会被跳过并且不会被跟踪。当存在此选项时，即使主心跳文件不存在，也会被跟踪。
	// 要在额外的心跳上设置此标志，请使用'is_unsaved_entity' json键。
	flags.Bool(
		"is-unsaved-entity",
		false,
		"Normally files that don't exist on disk are skipped and not tracked. When this option is present,"+
			" the main heartbeat file will be tracked even if it doesn't exist. To set this flag on"+
			" extra heartbeats, use the 'is_unsaved_entity' json key.")
	// 您的wakatime api密钥；默认使用~/.wakatime.cfg中的api_key。
	flags.String("key", "", "Your wakatime api key; uses api_key from ~/.wakatime.cfg by default.")
	// 可选的语言名称。如果有效，则优先于自动检测的语言。
	flags.String("language", "", "Optional language name. If valid, takes priority over auto-detected language.")
	// 可选的行号。这是当前正在编辑的行。
	flags.Int("lineno", 0, "Optional line number. This is the current line being edited.")
	// 文件中的可选行数。通常，这会自动检测，但可以手动提供以提高性能、准确性或使用--local-file时。
	flags.Int(
		"lines-in-file",
		0,
		"Optional lines in the file. Normally, this is detected automatically but"+
			" can be provided manually for performance, accuracy, or when using --local-file.")
	// 当前文件中自上次心跳以来添加的可选行数。
	flags.Int("line-additions", 0, "Optional number of lines added since last heartbeat in the current file.")
	// 当前文件中自上次心跳以来删除的可选行数。
	flags.Int("line-deletions", 0, "Optional number of lines deleted since last heartbeat in the current file.")
	// 心跳的本地文件的绝对路径。当--entity是远程文件时，此本地文件将用于统计数据，并且仅将--entity的值与心跳一起发送。
	flags.String(
		"local-file",
		"",
		"Absolute path to local file for the heartbeat. When --entity is a"+
			" remote file, this local file will be used for stats and just"+
			" the value of --entity is sent with the heartbeat.",
	)
	// 可选的日志文件。默认为'~/.wakatime/wakatime.log'。
	flags.String("log-file", "", "Optional log file. Defaults to '~/.wakatime/wakatime.log'.")
	// （已弃用）可选的日志文件。默认为'~/.wakatime/wakatime.log'。
	flags.String("logfile", "", "(deprecated) Optional log file. Defaults to '~/.wakatime/wakatime.log'.")
	// 如果启用，日志将转到stdout。将覆盖日志文件配置。
	flags.Bool("log-to-stdout", false, "If enabled, logs will go to stdout. Will overwrite logfile configs.")
	// 设置时，在'~/.wakatime/metrics'文件夹中收集使用情况指标。默认为false。
	flags.Bool(
		"metrics",
		false,
		"When set, collects metrics usage in '~/.wakatime/metrics' folder. Defaults to false.",
	)
	// 禁用HTTPS请求的SSL证书验证。默认情况下，SSL证书会被验证。
	flags.Bool(
		"no-ssl-verify",
		false,
		"Disables SSL certificate verification for HTTPS requests. By default,"+
			" SSL certificates are verified.",
	)
	// （内部）指定离线队列文件，将使用它而不是默认文件。
	flags.String(
		"offline-queue-file",
		"",
		"(internal) Specify an offline queue file, which will be used instead of the default one.",
	)
	// （内部）指定旧版离线队列文件，将使用它而不是默认文件。
	flags.String(
		"offline-queue-file-legacy",
		"",
		"(internal) Specify the legacy offline queue file, which will be used instead of the default one.",
	)
	// 格式化输出。可以是"text"、"json"或"raw-json"。默认为"text"。
	flags.String(
		"output",
		"",
		"Format output. Can be \"text\", \"json\" or \"raw-json\". Defaults to \"text\".",
	)
	// 可选的文本编辑器插件名称和版本，用于User-Agent头。
	flags.String("plugin", "", "Optional text editor plugin name and version for User-Agent header.")
	// 将离线心跳打印到stdout。
	flags.Int("print-offline-heartbeats", offline.PrintMaxDefault, "Prints offline heartbeats to stdout.")
	// 覆盖自动检测的项目。使用--alternate-project提供一个备用项目，如果无法自动检测到项目。
	flags.String("project", "", "Override auto-detected project."+
		" Use --alternate-project to supply a fallback project if one can't be auto-detected.")
	// 可选的工作区路径。通常用于隐藏项目文件夹，或者当无法自动检测到项目根文件夹时。
	flags.String(
		"project-folder",
		"",
		"Optional workspace path. Usually used when hiding the project folder, or when a project"+
			" root folder can't be auto detected.")
	// 可选的代理配置。支持HTTPS SOCKS和NTLM代理。例如：'https://user:pass@host:port'或'socks5://user:pass@host:port'或'domain\\user:pass'。
	flags.String(
		"proxy",
		"",
		"Optional proxy configuration. Supports HTTPS SOCKS and NTLM proxies."+
			" For example: 'https://user:pass@host:port' or 'socks5://user:pass@host:port'"+
			" or 'domain\\user:pass'",
	)
	// 当启用--verbose或debug时，还会在发生任何错误时发送诊断，而不仅仅是崩溃。
	flags.Bool(
		"send-diagnostics-on-errors",
		false,
		"When --verbose or debug enabled, also sends diagnostics on any error not just crashes.",
	)
	// 覆盖捆绑的CA证书文件。默认情况下，使用系统ca证书。
	flags.String(
		"ssl-certs-file",
		"",
		"Override the bundled CA certs file. By default, uses"+
			" system ca certs.",
	)
	// 从本地~/.wakatime/offline_heartbeats.bdb bolt文件同步离线活动到您的WakaTime仪表板，然后退出。
	// 可以是零或正整数。默认为1000，意味着在在线发送心跳后，所有排队的离线心跳都会发送到WakaTime API，最多限制为1000。
	// 零同步所有离线心跳。可以在没有--entity的情况下使用，仅同步离线活动而不生成新的心跳。
	flags.Int(
		"sync-offline-activity",
		offline.SyncMaxDefault,
		fmt.Sprintf("Amount of offline activity to sync from your local ~/.wakatime/offline_heartbeats.bdb bolt"+
			" file to your WakaTime Dashboard before exiting. Can be zero or"+
			" a positive integer. Defaults to %d, meaning after sending a heartbeat"+
			" while online, all queued offline heartbeats are sent to WakaTime API, up"+
			" to a limit of 1000. Zero syncs all offline heartbeats. Can be used"+
			" without --entity to only sync offline activity without generating"+
			" new heartbeats.", offline.SyncMaxDefault),
	)
	// 打印离线数据库中的心跳数量，然后退出。
	flags.Bool("offline-count", false, "Prints the number of heartbeats in the offline db, then exits.")
	// 发送心跳到api时等待的秒数。默认为60秒。
	flags.Int(
		"timeout",
		api.DefaultTimeoutSecs,
		fmt.Sprintf(
			"Number of seconds to wait when sending heartbeats to api. Defaults to %d seconds.", api.DefaultTimeoutSecs),
	)
	// 可选的浮点unix纪元时间戳。默认使用当前时间。
	flags.Float64("time", 0, "Optional floating-point unix epoch timestamp. Uses current time by default.")
	// 打印今天的仪表板时间，然后退出。
	flags.Bool("today", false, "Prints dashboard time for today, then exits.")
	// 当可选地与--today一起使用时，导致输出显示今天的总代码时间而没有类别。默认为false。
	flags.String("today-hide-categories", "", "When optionally included with --today, causes output to"+
		" show total code time today without categories. Defaults to false.")
	// 打印给定目标id今天的时间，然后退出。访问wakatime.com/api/v1/users/current/goals以找到您的目标id。
	flags.String(
		"today-goal",
		"",
		"Prints time for the given goal id today, then exits"+
			" Visit wakatime.com/api/v1/users/current/goals to find your goal id.")
	// （内部）打印wakatime-cli用户代理，因为它将被发送到api，然后退出。
	flags.Bool(
		"user-agent",
		false,
		"(internal) Prints the wakatime-cli useragent, as it will be sent to the api, then exits.",
	)
	// 在日志文件中打开调试消息，并在发生崩溃时发送诊断。
	flags.Bool("verbose", false, "Turns on debug messages in log file, and sends diagnostics if a crash occurs.")
	// 打印wakatime-cli版本号，然后退出。
	flags.Bool("version", false, "Prints the wakatime-cli version number, then exits.")
	// 设置时，告诉api此心跳是由写入文件触发的。
	flags.Bool("write", false, "When set, tells api this heartbeat was triggered from writing to a file.")

	// 隐藏已弃用的标志
	_ = flags.MarkHidden("apiurl")
	_ = flags.MarkHidden("disableoffline")
	_ = flags.MarkHidden("file")
	_ = flags.MarkHidden("hide-filenames")
	_ = flags.MarkHidden("hidefilenames")
	_ = flags.MarkHidden("logfile")

	// 隐藏内部标志
	_ = flags.MarkHidden("offline-queue-file")
	_ = flags.MarkHidden("offline-queue-file-legacy")
	_ = flags.MarkHidden("user-agent")

	err := v.BindPFlags(flags)
	if err != nil {
		log.Fatalf("failed to bind cobra flags to viper: %s", err)
	}
}

// Execute 将所有子命令添加到根命令并适当设置标志。
// 这由 main.main() 调用。它只需要对 rootCmd 发生一次。
func Execute() {
	if err := NewRootCMD().Execute(); err != nil {
		log.Fatalf("failed to run wakatime-cli: %s", err)
	}
}
