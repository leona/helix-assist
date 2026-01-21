interface Config {
  host: string;
  port: number;
  database: string;
  credentials: {
    username: string;
    password: string;
  };
}

function createConnection(config: Config) {
  return `Connecting to ${config.host}:${config.port}`;
}

// Cursor in middle of line after opening brace
const config: Config = {<CURSOR>};

// Expected: completion should suggest object properties
