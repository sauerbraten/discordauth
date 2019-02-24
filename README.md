# Maître d′

An auth server for [Cube 2: Sauerbraten](http://sauerbraten.org/).


## Why?

This auth server is usable by Sauerbraten game servers like a master server, except that it only understands auth related parts of the protocol, i.e.: `reqauth` and `confauth`. It responds the same way the masterserver at sauerbraten.org would respond to gauth requests (`chalauth`, `succauth`, `failauth`).

It is intended to be used by game servers for a specific auth key domain. The game server decides what privileges, if any, to grant authenticated users of that domain.


## Adding auth keys

Since names and public keys are stored in an SQLite database, [`/cmd/addauth/`](/cmd/addauth/) contains code for a binary to add users to the database for convenience. Use it like this:

	/addauth pix +daf0f6c8cfbcdf98cc56a316da0702b5423514e98673f1a6