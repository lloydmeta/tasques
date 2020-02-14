use serde::Deserialize;
use std::time;
use tasques_client::apis::configuration::Configuration;
use tasques_client::apis::TasksApiClient;

#[derive(Clone, Debug, Deserialize)]
pub struct WorkerId(pub String);

#[derive(Deserialize)]
pub struct BasicAuthUser {
    pub name: String,
    pub password: String,
}

#[derive(Deserialize)]
pub struct TasquesServer {
    pub address: String,
    pub base_path: Option<String>,
    pub schemes: Option<Vec<String>>,
    pub auth: Option<BasicAuthUser>,
}

impl TasquesServer {
    pub fn into_client(self) -> TasksApiClient {
        let mut conf = Configuration::new();
        let mut api_path = self.address;

        if let Some(base_path) = self.base_path {
            api_path = format!("{}/{}", api_path, base_path);
        }
        conf.base_path = api_path;
        if let Some(auth) = self.auth {
            conf.basic_auth = Some((auth.name, Some(auth.password)))
        }
        TasksApiClient::new(conf)
    }
}

#[derive(Deserialize)]
pub struct TasquesWorker {
    pub id: WorkerId,
    pub claim_amount: u32,
    pub block_for: time::Duration,
    pub server: TasquesServer,
}
