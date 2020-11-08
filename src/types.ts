import { DataQuery, DataSourceJsonData } from '@grafana/data';

export interface ConsulQuery extends DataQuery {
  target: string;
  format?: string;
  type?: string;
  legendFormat?: string;
  columns?: string;
}

/**
 * These are options configured for each DataSource instance
 */
export interface MyDataSourceOptions extends DataSourceJsonData {
  consulAddr?: string;
}

/**
 * Value that is used in the backend, but never sent over HTTP to the frontend
 */
export interface MySecureJsonData {
  consulToken?: string;
}
