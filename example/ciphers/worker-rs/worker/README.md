## Worker-rs

An example of a worker loop. Completely because OpenAPI's generator still generates synchronous clients for
Reqwest, but we wrap the synchronous calls using Tokio.

Oh, also had to get rid of RC wrapping the config..