## API module

This holds API-related logic that are API framework-agnostic.

The models here should, as much as possible, have fields that are recursively simple types that easily marshall and
demarshal to JSON.

The only types from the domain package that should be re-used in this module are the newtypes. 