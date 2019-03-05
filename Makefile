all: bootstrap_db
	go get ./...
	go build ./cmd/maitred
	go build ./cmd/manage
	go build ./cmd/stats

clean:
	rm -f maitred.sqlite ./maitred ./manage ./stats
	if [ ! -f maitred.sqlite ];	then \
		sqlite3 maitred.sqlite < maitred.sqlite.schema && \
	fi \

bootstrap_db:
	if [ ! -f maitred.sqlite ];	then \
		sqlite3 maitred.sqlite "insert or ignore into users (name, pubkey, admin) values ('pix', '+6a3913aa4e4a7a45aa890d289dfe099ed7825a80927c2edb', 1);"; \
		sqlite3 maitred.sqlite "insert or ignore into users (name, pubkey) values ('miu', '+ae6b173130fb836b5260ae11e30bee912e1ab0b35916c843');"; \
		sqlite3 maitred.sqlite "insert or ignore into users (name, pubkey) values ('Frosty', '+99be4d5a3f77076eb20978797d43bd9ccf94b39e03a5cb2');"; \
		sqlite3 maitred.sqlite "insert or ignore into users (name, pubkey) values ('Tagger', '+a1e915b99c0a2cc5a1b3590377e04bdc490f46fab7a7fc00');"; \
		sqlite3 maitred.sqlite "insert or ignore into users (name, pubkey) values ('Ignis', '+1e67875ae6107de18f006bebc34e9e44f795e35e67e4d9ca');"; \
		sqlite3 maitred.sqlite "insert or ignore into users (name, pubkey) values ('Murrr', '-8836a84f75db88f28b4dda4394ccd3ad77a1a556cc58169f');"; \
		sqlite3 maitred.sqlite "insert or ignore into users (name, pubkey) values ('Redon', '-efa043131ca8e8f68ed98cfcc069ee2d8fac00f0a5b523f7');"; \
	fi \
