package utils

// // Remove any CAPC managed Affinity groups if owner references a deleted object.
// if deleted, err := csCtrlrUtils.IsOwnerDeleted(ctx, r.Client, capiMachine); err != nil {
// 	return ctrl.Result{}, err
// } else if deleted {
// 	if err := r.RemoveManagedAffinity(runner.Log, capiMachine, csMachine); err != nil {
// 		return ctrl.Result{}, errors.Wrap(err, "error encountered when removing affinity group")
// 	}
// }

// // RemoveManagedAffinity considers a machine's affinity management strategy and removes the created affinity group
// // if it exists.
// func (r *CloudStackMachineReconciler) RemoveManagedAffinity(
// 	log logr.Logger,
// 	capiMachine *capiv1.Machine,
// 	csMachine *infrav1.CloudStackMachine,
// ) error {

// 	ownerRef := csCtrlrUtils.GetManagementOwnerRef(capiMachine)
// 	if ownerRef == nil {
// 		return errors.Errorf("Could not find management owner reference for %s/%s", csMachine.Namespace, csMachine.Name)
// 	}
// 	name, err := csMachine.AffinityGroupName(capiMachine)
// 	if err != nil {
// 		return err
// 	}
// 	group := &cloud.AffinityGroup{Name: name}
// 	_ = r.CS.FetchAffinityGroup(group)
// 	if group.ID == "" { // Affinity group not found, must have been deleted.
// 		return nil
// 	}

// 	log.Info(fmt.Sprintf("Deleting affinity group '%s'", name))

// 	return r.CS.DeleteAffinityGroup(group)
// }
