# lulis

Lulis is a simple PoC putting together FFMPEG, ChatGPT, Eleven Labs and Replicate.com to create a 24/7 AI Live Stream

## Requirements

```yaml
- TWITCH_CHANNEL_NAME=${LULIS_TWITCH_CHANNEL_NAME}
- TWITCH_STREAM_KEY=${LULIS_TWITCH_STREAM_KEY}
- TWITCH_CLIENT_ID=${LULIS_TWITCH_CLIENT_ID}
- OPEN_AI_KEY=${LULIS_OPEN_AI_KEY}
- ELEVEN_LABS_KEY=${LULIS_ELEVEN_LABS_KEY}
- ELEVEN_LABS_VOICE_ID=${LULIS_ELEVEN_LABS_VOICE_ID}
- REPLICATE_KEY=${LULIS_REPLICATE_KEY}
- AWS_BUCKET_NAME=${LULIS_AWS_BUCKET_NAME}
- AWS_REGION=${LULIS_AWS_REGION}
- AWS_ACCESS_KEY_ID=${LULIS_AWS_ACCESS_KEY_ID}
- AWS_SECRET_ACCESS_KEY=${LULIS_AWS_SECRET_ACCESS_KEY}
```

Considering the following environment variables that we have to configure first, you can already see the list of services that we will have to prepare first:
- Twitch Account (https://twitchapps.com/tmi/ use this for TWITCH_CLIENT_ID)
- OpenAI Developer Account https://platform.openai.com/login?launch
- Eleven Labs https://elevenlabs.io/ with a Cloned Voice for the ELEVEN_LABS_VOICE_ID
- Replicate.com https://replicate.com/
- AWS S3 https://aws.amazon.com/pm/serv-s3

## Run

```bash
$ docker-compose up
```

## Test

```bash
$ go test ./...
```

No tests written yet 👹