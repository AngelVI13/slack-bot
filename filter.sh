cat slack-bot.log | jq '. | select( .user | startswith("angel") )'
