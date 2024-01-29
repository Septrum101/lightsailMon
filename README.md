# Lightsail Monitor
An AWS Lightsail monitor service that can auto change blocked IP.
## Feature
- Support message push when IP is changed via `PushPlus` or `Telegram Bot`.
- Support auto sync IP with `Cloudflare` and `Google Domain`
## How to use
refer:  [config.example.yml](release/config.example.yml)
```yml
LogLevel: warning # Log level: debug, info, warn, error, fatal, panic
Internal: 300 # Time to check the node connection (unit: second)
Timeout: 15 # Timeout for the tcp request (unit: second)
Concurrent: 20 # Max concurrent on nodes check

DDNS:
  Enable: true
  Provider: cloudflare
  Config:
    CLOUDFLARE_EMAIL: test@test.com
    CLOUDFLARE_API_KEY: YOUR_TOKEN
#  Provider: google
#  Config:
#    GOOGLEDOMAIN_USERNAME: username
#    GOOGLEDOMAIN_PASSWORD: password

Notify:
  Enable: false
  Provider: pushplus
  Config:
    PUSHPLUS_TOKEN: YOUR_TOKEN
#  Provider: telegram
#  Config:
#    TELEGRAM_CHATID: 123
#    TELEGRAM_TOKEN: YOUR_TOKEN

Nodes:
  - AccessKeyID: YOUR_AWS_AccessKeyID
    SecretAccessKey: YOUR_AWS_SecretAccessKey
    Region: ap-northeast-1 # AWS service endpoints, check https://docs.aws.amazon.com/general/latest/gr/rande.html for help
    InstanceName: Debian-1
    Network: tcp4 # The type of network (tcp4, tcp6)
    Domain: node1.test.com # The node domain
    Port: 8080 # The node port

  - AccessKeyID: YOUR_AWS_AccessKeyID
    SecretAccessKey: YOUR_AWS_SecretAccessKey
    Region: ap-northeast-1 # AWS service endpoints, check https://docs.aws.amazon.com/general/latest/gr/rande.html for help
    InstanceName: Debian-1
    Network: tcp4 # The type of network (tcp4, tcp6)
    Domain: node2.test.com # The node domain
    Port: 8080 # The node port
```
### Installation
#### Docker (recommend)
Rename `config.example.yml` to `config.yml` and fill with right params, and then run those command:
```shell
docker run -d -v .:/etc/LightsailMon
```
#### Others
Currently not supported
## Sponsors
Thanks to the open source project license provided by [Jetbrains](https://www.jetbrains.com/)