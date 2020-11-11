import { DataSourceWithBackend, getBackendSrv, getTemplateSrv, toDataQueryResponse } from '@grafana/runtime';
import { MyDataSourceOptions, ConsulQuery } from './types';
import {
  DataQueryRequest,
  DataQueryResponse,
  DataSourceInstanceSettings,
  LoadingState,
  MetricFindValue,
} from '@grafana/data';

import { Observable } from 'rxjs';
import { map } from 'rxjs/operators';
import _ from 'lodash';
import { DataQueryResponseData } from '@grafana/data/types/datasource';

export class DataSource extends DataSourceWithBackend<ConsulQuery, MyDataSourceOptions> {
  constructor(instanceSettings: DataSourceInstanceSettings<MyDataSourceOptions>) {
    super(instanceSettings);
  }

  query(options: DataQueryRequest<ConsulQuery>): Observable<DataQueryResponse> {
    for (const target of options.targets) {
      target.target = getTemplateSrv().replace(target.target, options.scopedVars);
    }

    let activeTargets: { [key: string]: any } = {};
    // const activeTargets: any[] = [];
    for (const target of options.targets) {
      if (target.hide) {
        continue;
      }
      activeTargets[target.refId] = target;
    }

    return super.query(options).pipe(
      map((rsp: DataQueryResponse) => {
        console.log('rsp', rsp);

        const finalRsp: DataQueryResponse = { data: [], state: LoadingState.Done };
        _.each(rsp.data, (data: any) => {
          console.log('data', data);
          const legendFormat = activeTargets[data.refId].legendFormat;
          if (!_.isEmpty(legendFormat)) {
            console.log('legendFormat', legendFormat);
            data.fields[1].name = this.renderTemplate(legendFormat, data.fields[1].labels);
            data.fields[1].labels = [];
            finalRsp.data.push(data);
          } else {
            finalRsp.data.push(data);
          }
          console.log('data2', data);
        });
        return finalRsp;
      })
    );
  }

  renderTemplate(aliasPattern: string, aliasData: string) {
    const aliasRegex = /\{\{\s*(.+?)\s*\}\}/g;
    return aliasPattern.replace(aliasRegex, function(match, g1) {
      if (aliasData[g1]) {
        return aliasData[g1];
      }
      return g1;
    });
  }

  async testDatasource() {
    return getBackendSrv()
      .fetch({
        url: '/api/tsdb/query',
        method: 'POST',
        data: {
          queries: [
            {
              type: 'test',
              refId: 'test',
              datasourceId: this.id,
            },
          ],
        },
      })
      .pipe(
        map((rsp: any) => {
          if (rsp.status === 200) {
            return { status: 'success', message: 'Data source is working', title: 'Success' };
          }
          return {
            status: 'error',
            message: 'Data source is not working: ' + rsp.message,
          };
        })
      );
  }

  metricFindQuery(query: string): Promise<MetricFindValue[]> {
    return getBackendSrv()
      .fetch({
        url: '/api/tsdb/query',
        method: 'POST',
        data: {
          queries: [
            {
              target: query,
              format: 'timeseries',
              type: 'keys',
              refId: 'keys',
              datasourceId: this.id,
            },
          ],
        },
      })
      .toPromise()
      .then((result: any) => {
        let resp: DataQueryResponse = toDataQueryResponse(result);

        let values: MetricFindValue[] = [];
        resp.data.forEach((data: DataQueryResponseData) => {
          values.push({ text: data.name, expandable: false });
        });

        return values;
      });
  }
}
