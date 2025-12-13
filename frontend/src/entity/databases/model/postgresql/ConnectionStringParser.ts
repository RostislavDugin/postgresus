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

export class ConnectionStringParser {
  /**
   * Parses a PostgreSQL connection string in various formats.
   *
   * Supported formats:
   * 1. Standard PostgreSQL URI: postgresql://user:pass@host:port/db
   * 2. Postgres URI: postgres://user:pass@host:port/db
   * 3. Supabase Direct: postgresql://postgres:pass@db.xxx.supabase.co:5432/postgres
   * 4. Supabase Pooler Session: postgres://postgres.ref:pass@aws-0-region.pooler.supabase.com:5432/postgres
   * 5. Supabase Pooler Transaction: same as above with port 6543
   * 6. JDBC: jdbc:postgresql://host:port/db?user=x&password=y
   * 7. Neon: postgresql://user:pass@ep-xxx.neon.tech/db
   * 8. Railway: postgresql://postgres:pass@xxx.railway.app:port/railway
   * 9. Render: postgresql://user:pass@xxx.render.com/db
   * 10. DigitalOcean: postgresql://user:pass@xxx.ondigitalocean.com:port/db?sslmode=require
   * 11. AWS RDS: postgresql://user:pass@xxx.rds.amazonaws.com:port/db
   * 12. Azure: postgresql://user@server:pass@xxx.postgres.database.azure.com:port/db?sslmode=require
   * 13. Heroku: postgres://user:pass@ec2-xxx.amazonaws.com:port/db
   * 14. CockroachDB: postgresql://user:pass@xxx.cockroachlabs.cloud:port/db?sslmode=verify-full
   * 15. With SSL params: postgresql://user:pass@host:port/db?sslmode=require
   * 16. libpq key-value: host=x port=5432 dbname=db user=u password=p
   */
  static parse(connectionString: string): ParseResult | ParseError {
    const trimmed = connectionString.trim();

    if (!trimmed) {
      return { error: 'Connection string is empty' };
    }

    // Try JDBC format first (starts with jdbc:)
    if (trimmed.startsWith('jdbc:postgresql://')) {
      return this.parseJdbc(trimmed);
    }

    // Try libpq key-value format (contains key=value pairs without ://)
    if (this.isLibpqFormat(trimmed)) {
      return this.parseLibpq(trimmed);
    }

    // Try URI format (postgresql:// or postgres://)
    if (trimmed.startsWith('postgresql://') || trimmed.startsWith('postgres://')) {
      return this.parseUri(trimmed);
    }

    return {
      error: 'Unrecognized connection string format',
    };
  }

  private static isLibpqFormat(str: string): boolean {
    // libpq format has key=value pairs separated by spaces
    // Must contain at least host= or dbname= to be considered libpq format
    return (
      !str.includes('://') &&
      (str.includes('host=') || str.includes('dbname=')) &&
      str.includes('=')
    );
  }

  private static parseUri(connectionString: string): ParseResult | ParseError {
    try {
      // Handle Azure format where username contains @: user@server:pass
      // Azure format: postgresql://user@servername:password@host:port/db
      const azureMatch = connectionString.match(
        /^postgres(?:ql)?:\/\/([^@:]+)@([^:]+):([^@]+)@([^:/?]+):?(\d+)?\/([^?]+)(?:\?(.*))?$/,
      );

      if (azureMatch) {
        const [, user, , password, host, port, database, queryString] = azureMatch;
        const isHttps = this.checkSslMode(queryString);

        return {
          host: host,
          port: port ? parseInt(port, 10) : 5432,
          username: decodeURIComponent(user),
          password: decodeURIComponent(password),
          database: decodeURIComponent(database),
          isHttps,
        };
      }

      // Standard URI parsing using URL API
      const url = new URL(connectionString);

      const host = url.hostname;
      const port = url.port ? parseInt(url.port, 10) : 5432;
      const username = decodeURIComponent(url.username);
      const password = decodeURIComponent(url.password);
      const database = decodeURIComponent(url.pathname.slice(1)); // Remove leading /
      const isHttps = this.checkSslMode(url.search);

      // Validate required fields
      if (!host) {
        return { error: 'Host is missing from connection string' };
      }

      if (!username) {
        return { error: 'Username is missing from connection string' };
      }

      if (!password) {
        return { error: 'Password is missing from connection string' };
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

  private static parseJdbc(connectionString: string): ParseResult | ParseError {
    try {
      // JDBC format: jdbc:postgresql://host:port/database?user=x&password=y
      const jdbcRegex = /^jdbc:postgresql:\/\/([^:/?]+):?(\d+)?\/([^?]+)(?:\?(.*))?$/;
      const match = connectionString.match(jdbcRegex);

      if (!match) {
        return {
          error:
            'Invalid JDBC connection string format. Expected: jdbc:postgresql://host:port/database?user=x&password=y',
          format: 'JDBC',
        };
      }

      const [, host, port, database, queryString] = match;

      if (!queryString) {
        return {
          error: 'JDBC connection string is missing query parameters (user and password)',
          format: 'JDBC',
        };
      }

      const params = new URLSearchParams(queryString);
      const username = params.get('user');
      const password = params.get('password');
      const isHttps = this.checkSslMode(queryString);

      if (!username) {
        return {
          error: 'Username (user parameter) is missing from JDBC connection string',
          format: 'JDBC',
        };
      }

      if (!password) {
        return {
          error: 'Password parameter is missing from JDBC connection string',
          format: 'JDBC',
        };
      }

      return {
        host,
        port: port ? parseInt(port, 10) : 5432,
        username: decodeURIComponent(username),
        password: decodeURIComponent(password),
        database: decodeURIComponent(database),
        isHttps,
      };
    } catch (e) {
      return {
        error: `Failed to parse JDBC connection string: ${(e as Error).message}`,
        format: 'JDBC',
      };
    }
  }

  private static parseLibpq(connectionString: string): ParseResult | ParseError {
    try {
      // libpq format: host=x port=5432 dbname=db user=u password=p
      // Values can be quoted with single quotes: password='my pass'
      const params: Record<string, string> = {};

      // Match key=value or key='quoted value'
      const regex = /(\w+)=(?:'([^']*)'|(\S+))/g;
      let match;

      while ((match = regex.exec(connectionString)) !== null) {
        const key = match[1];
        const value = match[2] !== undefined ? match[2] : match[3];
        params[key] = value;
      }

      const host = params['host'] || params['hostaddr'];
      const port = params['port'];
      const database = params['dbname'] || params['database'];
      const username = params['user'] || params['username'];
      const password = params['password'];
      const sslmode = params['sslmode'];

      if (!host) {
        return {
          error: 'Host is missing from connection string. Use host=hostname',
          format: 'libpq',
        };
      }

      if (!username) {
        return {
          error: 'Username is missing from connection string. Use user=username',
          format: 'libpq',
        };
      }

      if (!password) {
        return {
          error: 'Password is missing from connection string. Use password=yourpassword',
          format: 'libpq',
        };
      }

      if (!database) {
        return {
          error: 'Database name is missing from connection string. Use dbname=database',
          format: 'libpq',
        };
      }

      const isHttps = this.isSslEnabled(sslmode);

      return {
        host,
        port: port ? parseInt(port, 10) : 5432,
        username,
        password,
        database,
        isHttps,
      };
    } catch (e) {
      return {
        error: `Failed to parse libpq connection string: ${(e as Error).message}`,
        format: 'libpq',
      };
    }
  }

  private static checkSslMode(queryString: string | undefined | null): boolean {
    if (!queryString) return false;

    const params = new URLSearchParams(
      queryString.startsWith('?') ? queryString.slice(1) : queryString,
    );
    const sslmode = params.get('sslmode');

    return this.isSslEnabled(sslmode);
  }

  private static isSslEnabled(sslmode: string | null | undefined): boolean {
    if (!sslmode) return false;

    // These modes require SSL
    const sslModes = ['require', 'verify-ca', 'verify-full'];
    return sslModes.includes(sslmode.toLowerCase());
  }
}
