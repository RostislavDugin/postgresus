// Package mysqlfamily holds the grant reading and backup-privilege rules that MySQL and MariaDB
// share. Both speak the same GRANT syntax and both dump through the same tool family, so keeping
// one copy is what keeps the two engine packages from drifting apart.
//
// What stays in the engine packages is what genuinely differs: how active roles are read (MySQL
// expands them with a USING clause, MariaDB has a single active role and no such clause) and the
// version enums, which are distinct types.
package mysqlfamily
