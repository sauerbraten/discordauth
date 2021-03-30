.PHONY: all discordauth clean nuke_db

all: discordauth

discordauth:
	go build ./cmd/discordauth

# utility targets

clean:
	rm -f ./discordauth

nuke_db:
	if [ -f discordauth.sqlite ]; then rm discordauth.sqlite; fi
