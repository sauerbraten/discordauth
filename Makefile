.PHONY: all maitred manage stats clean rebuild_db

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

rebuild_db:
	if [ -f maitred.sqlite ]; then rm maitred.sqlite; fi
	for m in migrations/*.up.sql; do sqlite3 maitred.sqlite < "$$m"; done