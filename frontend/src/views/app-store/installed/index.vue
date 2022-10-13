<template>
    <div style="float: right; margin-bottom: 5px">
        <el-button @click="sync">{{ $t('app.sync') }}</el-button>
    </div>
    <ComplexTable :pagination-config="paginationConfig" :data="data" @search="search" v-loading="loading">
        <el-table-column :label="$t('app.name')" prop="name">
            <template #default="{ row }">
                {{ row.name }}
                <el-tag round effect="dark" v-if="row.canUpdate">{{ $t('app.canUpdate') }}</el-tag>
            </template>
        </el-table-column>
        <!-- <el-table-column :label="$t('app.description')" prop="description"></el-table-column> -->
        <el-table-column :label="$t('app.appName')" prop="app.name"></el-table-column>
        <el-table-column :label="$t('app.version')" prop="version"></el-table-column>
        <el-table-column :label="$t('app.backup')">
            <template #default="{ row }">
                <el-link :underline="false" @click="openBackups(row.id)" type="primary">
                    {{ $t('app.backup') }} ({{ row.backups.length }})
                </el-link>
            </template>
        </el-table-column>
        <el-table-column :label="$t('app.status')">
            <template #default="{ row }">
                <el-popover
                    v-if="row.status === 'Error'"
                    placement="bottom"
                    :width="400"
                    trigger="hover"
                    :content="row.message"
                >
                    <template #reference>
                        <el-tag type="error">{{ row.status }}</el-tag>
                    </template>
                </el-popover>
                <el-tag v-else>{{ row.status }}</el-tag>
            </template>
        </el-table-column>
        <el-table-column
            prop="createdAt"
            :label="$t('commons.table.date')"
            :formatter="dateFromat"
            show-overflow-tooltip
        />
        <fu-table-operations
            width="300px"
            :ellipsis="10"
            :buttons="buttons"
            :label="$t('commons.table.operate')"
            fixed="right"
            fix
        />
    </ComplexTable>
    <el-dialog v-model="open" :title="$t('commons.msg.operate')" :before-close="handleClose" width="30%">
        <el-alert
            v-if="operateReq.operate != 'update'"
            :title="getMsg(operateReq.operate)"
            type="warning"
            :closable="false"
            show-icon
        />
        <div v-else style="text-align: center">
            <p>{{ $t('app.versioneSelect') }}</p>
            <el-select v-model="operateReq.detailId">
                <el-option
                    v-for="(version, index) in versions"
                    :key="index"
                    :value="version.detailId"
                    :label="version.version"
                ></el-option>
            </el-select>
        </div>
        <template #footer>
            <span class="dialog-footer">
                <el-button @click="handleClose">{{ $t('commons.button.cancel') }}</el-button>
                <el-button type="primary" @click="operate" :disabled="versions == null">
                    {{ $t('commons.button.confirm') }}
                </el-button>
            </span>
        </template>
    </el-dialog>
    <Backups ref="backupRef" @close="search"></Backups>
</template>

<script lang="ts" setup>
import { GetAppInstalled, InstalledOp, SyncInstalledApp, GetAppUpdateVersions } from '@/api/modules/app';
import { onMounted, reactive, ref } from 'vue';
import ComplexTable from '@/components/complex-table/index.vue';
import { dateFromat } from '@/utils/util';
import i18n from '@/lang';
import { ElMessage } from 'element-plus';
import Backups from './backups.vue';
import { App } from '@/api/interface/app';

let data = ref<any>();
let loading = ref(false);
const paginationConfig = reactive({
    currentPage: 1,
    pageSize: 20,
    total: 0,
});
let open = ref(false);
let operateReq = reactive({
    installId: 0,
    operate: '',
    detailId: 0,
});
let versions = ref<App.VersionDetail[]>();
const backupRef = ref();

const sync = () => {
    loading.value = true;
    SyncInstalledApp()
        .then(() => {
            ElMessage.success(i18n.global.t('app.syncSuccess'));
            search();
        })
        .finally(() => {
            loading.value = false;
        });
};

const search = () => {
    const req = {
        page: paginationConfig.currentPage,
        pageSize: paginationConfig.pageSize,
    };

    GetAppInstalled(req).then((res) => {
        data.value = res.data.items;
        paginationConfig.total = res.data.total;
    });
};

const openOperate = (row: any, op: string) => {
    operateReq.installId = row.id;
    operateReq.operate = op;
    if (op == 'update') {
        GetAppUpdateVersions(row.id).then((res) => {
            versions.value = res.data;
            if (res.data != null && res.data.length > 0) {
                operateReq.detailId = res.data[0].detailId;
            }
            open.value = true;
        });
    } else {
        open.value = true;
    }
};

const operate = async () => {
    open.value = false;
    loading.value = true;
    await InstalledOp(operateReq)
        .then(() => {
            ElMessage.success(i18n.global.t('commons.msg.operationSuccess'));
            search();
        })
        .finally(() => {
            loading.value = false;
        });
};

const handleClose = () => {
    open.value = false;
};

const getMsg = (op: string) => {
    let tip = '';
    switch (op) {
        case 'up':
            tip = i18n.global.t('app.up');
            break;
        case 'down':
            tip = i18n.global.t('app.down');
            break;
        case 'restart':
            tip = i18n.global.t('app.restart');
            break;
        case 'delete':
            tip = i18n.global.t('app.deleteWarn');
            break;
        case 'sync':
            tip = i18n.global.t('app.sync');
            break;
        default:
    }
    return tip;
};

const buttons = [
    {
        label: i18n.global.t('app.sync'),
        click: (row: any) => {
            openOperate(row, 'sync');
        },
    },
    {
        label: i18n.global.t('app.update'),
        click: (row: any) => {
            openOperate(row, 'update');
        },
    },
    {
        label: i18n.global.t('app.restart'),
        click: (row: any) => {
            openOperate(row, 'restart');
        },
    },
    {
        label: i18n.global.t('app.up'),
        click: (row: any) => {
            openOperate(row, 'up');
        },
    },
    {
        label: i18n.global.t('app.down'),
        click: (row: any) => {
            openOperate(row, 'down');
        },
    },
    {
        label: i18n.global.t('app.delete'),
        click: (row: any) => {
            openOperate(row, 'delete');
        },
    },
];

const openBackups = (installId: number) => {
    let params = {
        appInstallId: installId,
    };
    backupRef.value.acceptParams(params);
};

onMounted(() => {
    search();
});
</script>

<style lang="scss">
.i-card {
    height: 60px;
    cursor: pointer;
    .content {
        .image {
            width: auto;
            height: auto;
        }
    }
}
.i-card:hover {
    border: 1px solid;
    border-color: $primary-color;
    z-index: 1;
}
</style>