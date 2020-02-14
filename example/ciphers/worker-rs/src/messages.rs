use serde::{Deserialize, Serialize};

use elasticsearch::{self, Error as EsError, *};
use serde::export::fmt::Error;
use serde::export::Formatter;
use std::fmt;

static MESSAGES_INDEX: &'static str = "messages_and_their_ciphers";

#[derive(Debug, Deserialize, Serialize)]
pub struct Message {
    pub plain: String,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct MessagePartialUpdate {
    #[serde(skip_serializing_if = "Option::is_none")]
    rot_1: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    rot_13: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    base_64: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    md_5: Option<String>,
}

impl MessagePartialUpdate {
    pub fn rot_1(s: String) -> MessagePartialUpdate {
        MessagePartialUpdate {
            rot_1: Some(s),
            rot_13: None,
            base_64: None,
            md_5: None,
        }
    }

    pub fn rot_13(s: String) -> MessagePartialUpdate {
        MessagePartialUpdate {
            rot_13: Some(s),
            rot_1: None,
            base_64: None,
            md_5: None,
        }
    }
    pub fn base_64(s: String) -> MessagePartialUpdate {
        MessagePartialUpdate {
            base_64: Some(s),
            rot_1: None,
            rot_13: None,
            md_5: None,
        }
    }
    pub fn md_5(s: String) -> MessagePartialUpdate {
        MessagePartialUpdate {
            md_5: Some(s),
            rot_1: None,
            rot_13: None,
            base_64: None,
        }
    }
}

pub struct MessagesRepo {
    pub es_client: Elasticsearch,
}

#[derive(Debug)]
pub enum MessagesRepoErr {
    EsError(EsError),
}

impl fmt::Display for MessagesRepoErr {
    fn fmt(&self, f: &mut Formatter<'_>) -> Result<(), Error> {
        f.write_fmt(format_args!("{:?}", self))
    }
}

impl From<EsError> for MessagesRepoErr {
    fn from(e: EsError) -> Self {
        MessagesRepoErr::EsError(e)
    }
}

impl MessagesRepo {
    pub async fn get(&self, id: &str) -> Result<Message, MessagesRepoErr> {
        let v = self
            .es_client
            .get(GetParts::IndexId(MESSAGES_INDEX, id))
            .send()
            .await?;
        Ok(v.read_body::<EsGetResponse>().await?._source)
    }

    pub async fn update(
        &self,
        id: &str,
        update: MessagePartialUpdate,
    ) -> Result<(), MessagesRepoErr> {
        self.es_client
            .update(UpdateParts::IndexId(MESSAGES_INDEX, id))
            .body(PartialUpdate { doc: update })
            .send()
            .await?;

        Ok(())
    }
}

#[derive(Serialize)]
struct PartialUpdate {
    doc: MessagePartialUpdate,
}

#[derive(Deserialize)]
struct EsGetResponse {
    _source: Message,
}
