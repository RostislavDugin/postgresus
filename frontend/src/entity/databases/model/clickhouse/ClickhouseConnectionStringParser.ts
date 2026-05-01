export type ParseResult = {
  host: string;
  port: number;
  username: string;
  password: string;
  database: string;
  isHttps: boolean;
};

export type ParseError = {
  error: string;
  format?: string;
};

const NATIVE_PORT = 9000;
const NATIVE_TLS_PORT = 9440;

export class ClickhouseConnectionStringParser {
  /**
   * Parses a ClickHouse connection string.
   *
   * Supported formats:
   * 1. Native URI:        clickhouse://user:pass@host:9000/db
   * 2. Native URI + TLS:  clickhouse://user:pass@host:9440/db?secure=true
   * 3. tcp:// alias:      tcp://user:pass@host:9000/db
   * 4. HTTP(S):           https://user:pass@host:8443/?database=db&secure=true
   *                       (parsed for convenience; v1 still connects via native TCP)
   * 5. Key-value:         host=x port=9000 database=db user=u password=p
   */
  static parse(connectionString: string): ParseResult | ParseError {
    const trimmed = connectionString.trim();

    if (!trimmed) {
      return { error: 'Connection string is empty' };
    }

    if (this.isKeyValueFormat(trimmed)) {
      return this.parseKeyValue(trimmed);
    }

    if (
      trimmed.startsWith('clickhouse://') ||
      trimmed.startsWith('clickhouse+native://') ||
      trimmed.startsWith('tcp://') ||
      trimmed.startsWith('https://') ||
      trimmed.startsWith('http://')
    ) {
      return this.parseUri(trimmed);
    }

    return {
      error:
        'Unrecognized connection string format. Expected one of: ' +
        'clickhouse://user:pass@host:port/db, tcp://..., https://..., or key-value pairs',
    };
  }

  private static isKeyValueFormat(str: string): boolean {
    return (
      !str.includes('://') &&
      (str.includes('host=') || str.includes('database=')) &&
      str.includes('=')
    );
  }

  private static parseUri(connectionString: string): ParseResult | ParseError {
    try {
      const url = new URL(connectionString);

      const isHttpScheme = url.protocol === 'http:' || url.protocol === 'https:';
      const secureParam = this.checkSecure(url.search);
      const isHttps = url.protocol === 'https:' || secureParam;

      const host = url.hostname;
      // HTTP(S) URLs commonly carry port 8123/8443. The backend speaks native
      // protocol, so any HTTP-scheme port is rewritten to its native
      // equivalent (9000 / 9440). Native URIs (clickhouse://, tcp://) keep
      // the user-specified port verbatim.
      let port: number;
      if (isHttpScheme) {
        port = isHttps ? NATIVE_TLS_PORT : NATIVE_PORT;
      } else {
        port = url.port ? parseInt(url.port, 10) : isHttps ? NATIVE_TLS_PORT : NATIVE_PORT;
      }

      const username = decodeURIComponent(url.username);
      const password = decodeURIComponent(url.password);

      // For HTTP(S) URLs the database is conventionally in ?database= rather
      // than the path; native URIs put it in the path.
      let database = decodeURIComponent(url.pathname.replace(/^\/+/, ''));
      if (isHttpScheme && !database) {
        database = new URLSearchParams(url.search).get('database') || '';
      }

      if (!host) {
        return { error: 'Host is missing from connection string' };
      }
      if (!username) {
        return { error: 'Username is missing from connection string' };
      }
      if (!database) {
        return { error: 'Database name is missing from connection string' };
      }

      return {
        host,
        port,
        username,
        password,
        database,
        isHttps,
      };
    } catch (e) {
      return {
        error: `Failed to parse connection string: ${(e as Error).message}`,
        format: 'URI',
      };
    }
  }

  private static parseKeyValue(connectionString: string): ParseResult | ParseError {
    try {
      const params: Record<string, string> = {};
      const regex = /(\w+)=(?:'([^']*)'|(\S+))/g;
      let match;

      while ((match = regex.exec(connectionString)) !== null) {
        const key = match[1];
        const value = match[2] !== undefined ? match[2] : match[3];
        params[key] = value;
      }

      const host = params['host'] || params['hostaddr'];
      const port = params['port'];
      const database = params['database'] || params['dbname'];
      const username = params['user'] || params['username'];
      const password = params['password'] || '';
      const secure = params['secure'] || params['tls'] || params['ssl'];

      if (!host) {
        return {
          error: 'Host is missing from connection string. Use host=hostname',
          format: 'key-value',
        };
      }
      if (!username) {
        return {
          error: 'Username is missing from connection string. Use user=username',
          format: 'key-value',
        };
      }
      if (!database) {
        return {
          error: 'Database name is missing from connection string. Use database=database',
          format: 'key-value',
        };
      }

      const isHttps = this.isFlagEnabled(secure);

      return {
        host,
        port: port ? parseInt(port, 10) : isHttps ? NATIVE_TLS_PORT : NATIVE_PORT,
        username,
        password,
        database,
        isHttps,
      };
    } catch (e) {
      return {
        error: `Failed to parse key-value connection string: ${(e as Error).message}`,
        format: 'key-value',
      };
    }
  }

  private static checkSecure(queryString: string | undefined | null): boolean {
    if (!queryString) return false;
    const params = new URLSearchParams(
      queryString.startsWith('?') ? queryString.slice(1) : queryString,
    );
    return (
      this.isFlagEnabled(params.get('secure')) ||
      this.isFlagEnabled(params.get('tls')) ||
      this.isFlagEnabled(params.get('ssl'))
    );
  }

  private static isFlagEnabled(value: string | null | undefined): boolean {
    if (!value) return false;
    const lowered = value.toLowerCase();
    return ['true', 'yes', '1', 'required'].includes(lowered);
  }
}
