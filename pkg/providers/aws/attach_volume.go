package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/emc-advanced-dev/unik/pkg/types"
	"github.com/layer-x/layerx-commons/lxerrors"
	"github.com/Sirupsen/logrus"
	"github.com/emc-advanced-dev/unik/pkg/providers/common"
)

func (p *AwsProvider) AttachVolume(id, instanceId, mntPoint string) error {
	volume, err := p.GetVolume(id)
	if err != nil {
		return lxerrors.New("retrieving volume "+id, err)
	}
	instance, err := p.GetInstance(instanceId)
	if err != nil {
		return lxerrors.New("retrieving instance "+id, err)
	}
	image, err := p.GetImage(instance.ImageId)
	if err != nil {
		return lxerrors.New("retrieving image for instance", err)
	}
	if err := common.VerifyMntsInput(p, image, map[string]string{mntPoint: id}); err != nil {
		return lxerrors.New("invalid mapping for volume", err)
	}
	deviceName := ""
	for _, mapping := range image.DeviceMappings {
		if mntPoint == mapping.MountPoint {
			deviceName = mapping.DeviceName
			break
		}
	}
	if deviceName == "" {
		logrus.WithFields(logrus.Fields{"image": image.Id, "mappings": image.DeviceMappings, "mount point": mntPoint}).Errorf("given mapping was not found for image")
		return lxerrors.New("no mapping found on image "+image.Id+" for mount point "+mntPoint, nil)
	}
	param := &ec2.AttachVolumeInput{
		VolumeId:   aws.String(volume.Id),
		InstanceId: aws.String(instance.Id),
		Device:     aws.String(deviceName),
	}
	_, err = p.newEC2().AttachVolume(param)
	if err != nil {
		return lxerrors.New("failed to attach volume "+volume.Id, err)
	}
	err = p.state.ModifyVolumes(func(volumes map[string]*types.Volume) error {
		volume, ok := volumes[volume.Id]
		if !ok {
			return lxerrors.New("no record of "+volume.Id+" in the state", nil)
		}
		volume.Attachment = instance.Id
		return nil
	})
	if err != nil {
		return lxerrors.New("modifying volume map in state", err)
	}
	err = p.state.Save()
	if err != nil {
		return lxerrors.New("saving volume to state", err)
	}
	return nil
}