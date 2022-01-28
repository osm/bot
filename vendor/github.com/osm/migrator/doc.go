/*
Migrator is a simple and easy to use database migration library.

Technical details

Each migration is executed within its own transaction.

Execution will stop immediately and a rollback will be issued
if there's any problem with the migration.

It will only rollback the failed migration and stop execution,
all successfully migrated versions will be kept.

All exported functions expects a working database connection.

The migrator functions makes sure that the connection to the database
is working before any migration scripts are executed.

It will load the migration scripts from the repository,
any errors on the steps above will be returned immediately.
*/
package migrator
