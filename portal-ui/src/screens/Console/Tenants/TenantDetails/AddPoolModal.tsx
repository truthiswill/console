import React, { useEffect, useState } from "react";
import get from "lodash/get";
import ModalWrapper from "../../Common/ModalWrapper/ModalWrapper";
import { Theme } from "@mui/material/styles";
import createStyles from "@mui/styles/createStyles";
import withStyles from "@mui/styles/withStyles";
import {
  formFieldStyles,
  modalStyleUtils,
} from "../../Common/FormComponents/common/styleLibrary";
import InputBoxWrapper from "../../Common/FormComponents/InputBoxWrapper/InputBoxWrapper";
import Grid from "@mui/material/Grid";
import { generatePoolName, niceBytes } from "../../../../common/utils";
import { Button, LinearProgress, SelectChangeEvent } from "@mui/material";
import api from "../../../../common/api";
import { IAddPoolRequest, ITenant } from "../ListTenants/types";
import { ErrorResponseHandler, IAffinityModel } from "../../../../common/types";
import { getDefaultAffinity } from "./utils";

import SelectWrapper from "../../Common/FormComponents/SelectWrapper/SelectWrapper";
import { IQuotaElement, IQuotas, Opts } from "../ListTenants/utils";
import { NewPoolIcon } from "../../../../icons";

interface IAddPoolProps {
  tenant: ITenant;
  classes: any;
  open: boolean;
  onClosePoolAndReload: (shouldReload: boolean) => void;
}

const styles = (theme: Theme) =>
  createStyles({
    buttonContainer: {
      textAlign: "right",
    },
    bottomContainer: {
      display: "flex",
      flexGrow: 1,
      alignItems: "center",
      margin: "auto",
      justifyContent: "center",
      "& div": {
        width: 150,
        "@media (max-width: 900px)": {
          flexFlow: "column",
        },
      },
    },
    factorElements: {
      display: "flex",
      justifyContent: "flex-start",
      marginLeft: 30,
    },
    sizeNumber: {
      fontSize: 35,
      fontWeight: 700,
      textAlign: "center",
    },
    sizeDescription: {
      fontSize: 14,
      color: "#777",
      textAlign: "center",
    },
    ...formFieldStyles,
    ...modalStyleUtils,
  });

const AddPoolModal = ({
  tenant,
  classes,
  open,
  onClosePoolAndReload,
}: IAddPoolProps) => {
  const [addSending, setAddSending] = useState<boolean>(false);
  const [numberOfNodes, setNumberOfNodes] = useState<number>(0);
  const [volumeSize, setVolumeSize] = useState<number>(0);
  const [volumesPerServer, setVolumesPerSever] = useState<number>(0);
  const [selectedStorageClass, setSelectedStorageClass] = useState<string>("");
  const [storageClasses, setStorageClasses] = useState<Opts[]>([]);

  const instanceCapacity: number = volumeSize * 1073741824 * volumesPerServer;
  const totalCapacity: number = instanceCapacity * numberOfNodes;

  useEffect(() => {
    setSelectedStorageClass("");

    setStorageClasses([]);
    api
      .invoke(
        "GET",
        `/api/v1/namespaces/${tenant.namespace}/resourcequotas/${tenant.namespace}-storagequota`
      )
      .then((res: IQuotas) => {
        const elements: IQuotaElement[] = get(res, "elements", []);

        const newStorage = elements.map((storageClass: any) => {
          const name = get(storageClass, "name", "").split(
            ".storageclass.storage.k8s.io/requests.storage"
          )[0];

          return { label: name, value: name };
        });

        setStorageClasses(newStorage);
        if (newStorage.length > 0) {
          setSelectedStorageClass(newStorage[0].value);
        }
      })
      .catch((err: ErrorResponseHandler) => {
        console.error(err);
      });
  }, [tenant]);

  return (
    <ModalWrapper
      onClose={() => onClosePoolAndReload(false)}
      modalOpen={open}
      title="Add Pool"
      titleIcon={<NewPoolIcon />}
    >
      <form
        noValidate
        autoComplete="off"
        onSubmit={(e: React.FormEvent<HTMLFormElement>) => {
          e.preventDefault();
          setAddSending(true);

          const poolName = generatePoolName(tenant.pools);

          const defaultAffinity: IAffinityModel = getDefaultAffinity(
            tenant.name,
            poolName
          );

          const data: IAddPoolRequest = {
            name: poolName,
            servers: numberOfNodes,
            volumes_per_server: volumesPerServer,
            volume_configuration: {
              size: volumeSize * 1073741824,
              storage_class_name: selectedStorageClass,
              labels: null,
            },
            affinity: defaultAffinity,
          };

          api
            .invoke(
              "POST",
              `/api/v1/namespaces/${tenant.namespace}/tenants/${tenant.name}/pools`,
              data
            )
            .then(() => {
              setAddSending(false);
              onClosePoolAndReload(true);
            })
            .catch((err: ErrorResponseHandler) => {
              setAddSending(false);
              // setDeleteError(err);
            });
        }}
      >
        <Grid item xs={12} className={classes.formFieldRow}>
          <InputBoxWrapper
            id="number_of_nodes"
            name="number_of_nodes"
            type="number"
            onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
              setNumberOfNodes(parseInt(e.target.value));
            }}
            label="Number of Nodes"
            value={numberOfNodes.toString(10)}
          />
        </Grid>
        <Grid item xs={12} className={classes.formFieldRow}>
          <InputBoxWrapper
            id="pool_size"
            name="pool_size"
            type="number"
            onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
              setVolumeSize(parseInt(e.target.value));
            }}
            label="Volume Size (Gi)"
            value={volumeSize.toString(10)}
          />
        </Grid>
        <Grid item xs={12} className={classes.formFieldRow}>
          <InputBoxWrapper
            id="volumes_per_sever"
            name="volumes_per_sever"
            type="number"
            onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
              setVolumesPerSever(parseInt(e.target.value));
            }}
            label="Volumes per Server"
            value={volumesPerServer.toString(10)}
          />
        </Grid>
        <Grid item xs={12} className={classes.formFieldRow}>
          <SelectWrapper
            id="storage_class"
            name="storage_class"
            onChange={(e: SelectChangeEvent<string>) => {
              setSelectedStorageClass(e.target.value as string);
            }}
            label="Storage Class"
            value={selectedStorageClass}
            options={storageClasses}
            disabled={storageClasses.length < 1}
          />
        </Grid>

        <Grid item xs={12} className={classes.bottomContainer}>
          <div className={classes.factorElements}>
            <div>
              <div className={classes.sizeNumber}>
                {niceBytes(instanceCapacity.toString(10))}
              </div>
              <div className={classes.sizeDescription}>Instance Capacity</div>
            </div>
            <div>
              <div className={classes.sizeNumber}>
                {niceBytes(totalCapacity.toString(10))}
              </div>
              <div className={classes.sizeDescription}>Total Capacity</div>
            </div>
          </div>
        </Grid>
        <Grid item xs={12} className={classes.modalButtonBar}>
          <Button
            type="button"
            variant="outlined"
            color="primary"
            disabled={addSending}
            onClick={() => onClosePoolAndReload(false)}
          >
            Cancel
          </Button>
          <Button
            type="submit"
            variant="contained"
            color="primary"
            disabled={addSending}
          >
            Save
          </Button>
        </Grid>
        {addSending && (
          <Grid item xs={12}>
            <LinearProgress />
          </Grid>
        )}
      </form>
    </ModalWrapper>
  );
};

export default withStyles(styles)(AddPoolModal);
