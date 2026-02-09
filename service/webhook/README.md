# Webhook

[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat)](https://pkg.go.dev/github.com/nikoksr/notify/service/webhook)

## Prerequisites
To use the webhook command, you will need:
- Obtain webook URL from any service
- You need to manually generate message in JSON format expected by target service. webhook example:
```json
{"content": "Hey there! It's notify"}
```

## Usage

### Basic Usage

```go
package main

import (
	"context"
	"log"

	"github.com/nikoksr/notify"
	"github.com/nikoksr/notify/service/webhook"
)

func main() {

	url := "https://discord.com/api/webhooks/"
	timeout := 5

	webhookSvc := webhook.New(url, timeout)

	message := `
	{
		"content": "test message"
	}
	`

	notifier := notify.New()
	notifier.UseServices(webhookSvc)

	// Subject is ignored, so you do not need to set it
	if err := notifier.Send(context.Background(), "", message); err != nil {
		log.Fatalf("failed to send notification: %s", err.Error())
	}

	log.Println("notification sent")
}
```