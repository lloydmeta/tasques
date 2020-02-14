use serde::Deserialize;
use tasques_worker::config::*;

#[derive(Deserialize)]
pub struct AppConfig {
    pub cipher_worker: CipherWorker,
    pub elasticsearch: Elasticsearch,
}

#[derive(Deserialize)]
pub struct CipherWorker {
    pub queue: String,
    pub tasques_worker: TasquesWorker,
}

#[derive(Deserialize)]
pub struct Elasticsearch {
    pub address: String,
    pub user: Option<BasicAuthUser>,
}
