use std::rc::Rc;

use super::configuration::Configuration;

pub struct APIClient {
    tasks_api: Box<dyn crate::apis::TasksApi>,
}

impl APIClient {
    pub fn new(configuration: Configuration) -> APIClient {
        APIClient {
            tasks_api: Box::new(crate::apis::TasksApiClient::new(configuration)),
        }
    }

    pub fn tasks_api(&self) -> &dyn crate::apis::TasksApi {
        self.tasks_api.as_ref()
    }
}
