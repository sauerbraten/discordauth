all: bootstrap_db
	go get ./...
	go build ./cmd/maitred
	go build ./cmd/manage
	go build ./cmd/stats

clean:
	rm -f maitred.sqlite ./maitred ./manage ./stats

bootstrap_db:
	if [ ! -f maitred.sqlite ];	then \
		sqlite3 maitred.sqlite < maitred.sqlite.schema && \
		sqlite3 maitred.sqlite "insert into users (name, pubkey, admin) values ('pix', '+6a3913aa4e4a7a45aa890d289dfe099ed7825a80927c2edb', 1);"; \
		sqlite3 maitred.sqlite "insert into users (name, pubkey) values ('miu', '+ae6b173130fb836b5260ae11e30bee912e1ab0b35916c843');"; \
		sqlite3 maitred.sqlite "insert into users (name, pubkey) values ('Frosty', '+99be4d5a3f77076eb20978797d43bd9ccf94b39e03a5cb2');"; \
		sqlite3 maitred.sqlite "insert into users (name, pubkey) values ('Tagger', '+a1e915b99c0a2cc5a1b3590377e04bdc490f46fab7a7fc00');"; \
	fi \
