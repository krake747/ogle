# ogle

[![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/ma-tf/ogle)](https://pkg.go.dev/github.com/ma-tf/ogle)
[![Go Report Card](https://goreportcard.com/badge/github.com/ma-tf/ogle)](https://goreportcard.com/report/github.com/ma-tf/ogle)
[![GitHub License](https://img.shields.io/github/license/ma-tf/ogle)](https://github.com/ma-tf/ogle/blob/master/COPYING)

```txt
       , ·. ,.-·~·.,   ‘              ,.-·^*ª'` ·,                 ,.  '                      _,.,  °    
      /  ·'´,.-·-.,   `,'‚           .·´ ,·'´:¯'`·,  '\‘            /   ';\               ,.·'´  ,. ,  `;\ '  
     /  .'´\:::::::'\   '\ °       ,´  ,'\:::::::::\,.·\'         ,'   ,'::'\            .´   ;´:::::\`'´ \'\  
  ,·'  ,'::::\:;:-·-:';  ';\‚      /   /:::\;·'´¯'`·;\:::\°      ,'    ;:::';'          /   ,'::\::::::\:::\:' 
 ;.   ';:::;´       ,'  ,':'\‚    ;   ;:::;'          '\;:·´      ';   ,':::;'          ;   ;:;:-·'~^ª*';\'´   
  ';   ;::;       ,'´ .'´\::';‚  ';   ;::/      ,·´¯';  °        ;  ,':::;' '          ;  ,.-·:*'´¨'`*´\::\ '  
  ';   ':;:   ,.·´,.·´::::\;'°  ';   '·;'   ,.·´,    ;'\         ,'  ,'::;'            ;   ;\::::::::::::'\;'   
   \·,   `*´,.·'´::::::;·´     \'·.    `'´,.·:´';   ;::\'       ;  ';_:,.-·´';\‘     ;  ;'_\_:;:: -·^*';\   
    \\:¯::\:::::::;:·´         '\::\¯::::::::';   ;::'; ‘     ',   _,.-·'´:\:\‘    ';    ,  ,. -·:*'´:\:'\° 
     `\:::::\;::·'´  °            `·:\:::;:·´';.·´\::;'         \¨:::::::::::\';     \`*´ ¯\:::::::::::\;' '
         ¯                           ¯      \::::\;'‚          '\;::_;:-·'´‘         \:::::\;::-·^*'´     
          ‘                                    '\:·´'              '¨                    `*´¯              
```

*ogle* is a terminal UI for observing and operating Docker Compose projects — no setup required.

![ogle Dashboard](docs/assets/ogle-dashboard.png)

## Requirements

- Go 1.26+ (to build from source)
- Docker daemon (for log streaming and service actions)
- A Docker Compose file (auto-discovered or specified with `-f`)

## Installation

```sh
go install github.com/ma-tf/ogle@latest
```

Or download a pre-built binary from the [releases page](https://github.com/ma-tf/ogle/releases).

## Quick Start

```sh
# Auto-discover compose.yaml in current directory
ogle

# Specify a compose file explicitly
ogle -f docker-compose.yml
```
