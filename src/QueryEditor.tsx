import React, { PureComponent } from 'react';
import { InlineFormLabel, Select } from '@grafana/ui';
import { QueryEditorProps, SelectableValue } from '@grafana/data';
import { DataSource } from './DataSource';
import { ConsulQuery, MyDataSourceOptions } from './types';

type Props = QueryEditorProps<DataSource, ConsulQuery, MyDataSourceOptions>;

const FORMAT_OPTIONS: Array<SelectableValue<string>> = [
  { label: 'Time series', value: 'timeseries' },
  { label: 'Table', value: 'table' },
];

const TYPE_OPTIONS: Array<SelectableValue<string>> = [
  { label: 'get value', value: 'get' },
  { label: 'get direct subkeys', value: 'keys' },
  { label: 'get subkeys as tags', value: 'tags' },
  { label: 'get subkeys recursive as tags', value: 'tagsrec' },
];

interface State {
  target: string;
  formatOption: SelectableValue<string>;
  typeOption: SelectableValue<string>;
  legendFormat?: string;
  columns?: string;
}

export class QueryEditor extends PureComponent<Props, State> {
  // Query target to be modified and used for queries
  query: ConsulQuery;

  constructor(props: Props) {
    super(props);
    const defaultQuery: Partial<ConsulQuery> = {
      target: '',
      format: 'timeseries',
      type: 'get',
      legendFormat: '',
      columns: '',
    };
    const query = Object.assign({}, defaultQuery, props.query);
    this.query = query;
    // Query target properties that are fully controlled inputs
    this.state = {
      target: query.target,
      // Fully controlled text inputs
      legendFormat: query.legendFormat,
      // Select options
      formatOption: FORMAT_OPTIONS.find(option => option.value === query.format) || FORMAT_OPTIONS[0],
      // Select options
      typeOption: TYPE_OPTIONS.find(option => option.value === query.type) || TYPE_OPTIONS[0],

      columns: query.columns,
    };
  }

  onTargetChanged = (e: React.SyntheticEvent<HTMLInputElement>) => {
    const target = e.currentTarget.value;
    this.query.target = target;
    this.setState({ target: target }, this.onRunQuery);
  };

  onFormatChange = (option: SelectableValue<string>) => {
    this.query.format = option.value;
    this.setState({ formatOption: option }, this.onRunQuery);
  };

  onTypeChange = (option: SelectableValue<string>) => {
    this.query.type = option.value;
    this.setState({ typeOption: option }, this.onRunQuery);
  };

  onLegendChange = (e: React.SyntheticEvent<HTMLInputElement>) => {
    const legendFormat = e.currentTarget.value;
    this.query.legendFormat = legendFormat;
    this.setState({ legendFormat }, this.onRunQuery);
  };

  onColumnsChange = (e: React.SyntheticEvent<HTMLInputElement>) => {
    const columns = e.currentTarget.value;
    this.query.columns = columns;
    this.setState({ columns }, this.onRunQuery);
  };

  onRunQuery = () => {
    const { query } = this;
    this.props.onChange(query);
    this.props.onRunQuery();
  };

  render() {
    const { target, formatOption, typeOption, legendFormat, columns } = this.state;

    return (
      <div>
        <div className="gf-form">
          <input
            type="text"
            className="gf-form-input"
            placeholder="query"
            value={target}
            onChange={this.onTargetChanged}
            onBlur={this.onRunQuery}
          />
        </div>

        <div className="gf-form-inline">
          <div className="gf-form-label width-7">Format</div>
          <Select
            width={16}
            isSearchable={false}
            options={FORMAT_OPTIONS}
            onChange={this.onFormatChange}
            value={formatOption}
          />

          {formatOption.value === 'timeseries' ? (
            <div className="gf-form">
              <div className="gf-form-label width-7">Type</div>
              <Select
                width={40}
                isSearchable={false}
                options={TYPE_OPTIONS}
                onChange={this.onTypeChange}
                value={typeOption}
              />
            </div>
          ) : null}

          {formatOption.value === 'timeseries' ? (
            <div className="gf-form">
              <InlineFormLabel
                width={7}
                tooltip="Controls the name of the time series, using name or pattern. For example
                {{hostname}} will be replaced with label value for the label hostname."
              >
                Legend
              </InlineFormLabel>
              <input
                type="text"
                className="gf-form-input"
                placeholder=""
                value={legendFormat}
                onChange={this.onLegendChange}
                onBlur={this.onRunQuery}
              />
            </div>
          ) : null}

          {formatOption.value === 'table' ? (
            <div className="gf-form">
              <InlineFormLabel width={7} tooltip="Comma-separated list of Consul keys which should be used as columns.">
                Columns
              </InlineFormLabel>
              <input
                type="text"
                className="gf-form-input"
                placeholder=""
                value={columns}
                onChange={this.onColumnsChange}
                onBlur={this.onRunQuery}
              />
            </div>
          ) : null}
        </div>
      </div>
    );
  }
}
