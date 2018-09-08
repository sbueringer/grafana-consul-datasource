// import { beforeEach, describe, expect, it } from './lib/common';
import TemplateSrvStub from './lib/TemplateSrvStub';
import {Datasource, QueryCtrl} from '../src/module';
import q from 'q';
import {beforeEach, describe, expect, it} from "./lib/common";

describe('ConsulDatasource', () => {
    const ctx: any = {
        templateSrv: new TemplateSrvStub(),
    };

    beforeEach(() => {
        ctx.qc = new QueryCtrl(null, null, ctx.templateSrv);
    });

    it('check default values', (done) => {
        let cq = ctx.qc;
        expect(cq.formats).toHaveLength(2);
        expect(cq.target.format).toBe("timeseries");
        expect(cq.target.type).toBe("get");
        done()
    });

});
