.PHONY: all maitred manage stats clean nuke_db

all: maitred manage stats

maitred:
	go build ./cmd/maitred

manage:
	go build ./cmd/manage

stats:
	go build ./cmd/stats


# utility targets

clean:
	rm -f ./maitred ./manage ./stats

nuke_db:
	if [ -f maitred.sqlite ]; then rm maitred.sqlite; fi
