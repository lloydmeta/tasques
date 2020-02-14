use crate::cipher_handler::CipherHandler;
use crate::messages::MessagesRepo;
use config::AppConfig;
use elasticsearch::auth::Credentials;
use elasticsearch::http::transport::*;
use elasticsearch::Elasticsearch;
use hocon::HoconLoader;
use std::collections::hash_map::RandomState;
use std::collections::HashMap;
use std::error::Error;
use structopt::StructOpt;
use tasques_worker::config::WorkerId;
use tasques_worker::work_loop::{QueuesToHandler, TaskHandler, WorkLoop};
use url::Url;

mod cipher_handler;
mod config;
mod messages;

#[tokio::main]
async fn main() -> Result<(), Box<dyn Error>> {
    pretty_env_logger::init();
    let opt: Opt = Opt::from_args();

    let conf = {
        let mut t: AppConfig = HoconLoader::new()
            .load_file(opt.config)?
            .hocon()?
            .resolve()?;
        if let Some(command_line_worker_id) = opt.worker_id {
            t.cipher_worker.tasques_worker.id = WorkerId(command_line_worker_id);
        }
        t
    };

    // A reqwest HTTP client and default parameters.
    // The builder includes the base node url (http://localhost:9200).
    let es_client = {
        let url = Url::parse(&conf.elasticsearch.address)?;
        let conn_pool = SingleNodeConnectionPool::new(url);
        let mut transport_builder = TransportBuilder::new(conn_pool);
        if let Some(auth) = conf.elasticsearch.user {
            transport_builder =
                transport_builder.auth(Credentials::Basic(auth.name, auth.password));
        }
        Elasticsearch::new(transport_builder.build()?)
    };

    let messages_repo = MessagesRepo {
        es_client: es_client,
    };
    let cipher_handler = CipherHandler {
        messages_repo: messages_repo,
    };

    let mut queues_to_handlers: HashMap<String, Box<dyn TaskHandler>, RandomState> = HashMap::new();
    let _ = queues_to_handlers.insert(conf.cipher_worker.queue, Box::new(cipher_handler));

    let work_loop = WorkLoop::build(
        conf.cipher_worker.tasques_worker,
        QueuesToHandler(queues_to_handlers),
    );

    Ok(work_loop.run().await?)
}

#[derive(Debug, StructOpt)]
#[structopt(name = "cipher-worker", about = "A Tasque worker example in rust")]
struct Opt {
    /// Worker Id, if not specified reads from config
    // short and long flags (-d, --debug) will be deduced from the field's name
    #[structopt(short = "i", long = "worker-id")]
    worker_id: Option<String>,

    /// Where to read config from
    // we don't want to name it "speed", need to look smart
    #[structopt(short, long, default_value = "./config/reference.conf")]
    config: String,
}
