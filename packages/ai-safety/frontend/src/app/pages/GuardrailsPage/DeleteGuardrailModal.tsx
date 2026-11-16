import React from 'react';
import { Alert, Button, Modal, ModalBody, ModalFooter, ModalHeader } from '@patternfly/react-core';
import { useNotification } from '~/app/hooks/useNotification';
import { deleteGuardrailDeployment } from '~/app/api/guardrails';
import { GuardrailDeployment } from '~/app/types';

type DeleteGuardrailModalProps = {
  deployment: GuardrailDeployment | null;
  onClose: () => void;
  onDeleted: () => void;
};

const DeleteGuardrailModal: React.FC<DeleteGuardrailModalProps> = ({
  deployment,
  onClose,
  onDeleted,
}) => {
  const notification = useNotification();
  const [isSubmitting, setIsSubmitting] = React.useState(false);
  const [error, setError] = React.useState<Error | null>(null);

  React.useEffect(() => {
    if (deployment) {
      setError(null);
    }
  }, [deployment]);

  const onSubmit = React.useCallback(async () => {
    if (!deployment) {
      return;
    }
    setIsSubmitting(true);
    setError(null);
    try {
      await deleteGuardrailDeployment('', deployment.namespace, deployment.name);
      notification.success(
        'Deployment deleted',
        `Guardrails deployment ${deployment.name} was deleted from namespace ${deployment.namespace}.`,
      );
      onDeleted();
    } catch (e) {
      setError(e instanceof Error ? e : new Error(String(e)));
    } finally {
      setIsSubmitting(false);
    }
  }, [deployment, notification, onDeleted]);

  if (!deployment) {
    return null;
  }

  return (
    <Modal
      isOpen
      onClose={onClose}
      variant="small"
      aria-label="Delete guardrails deployment"
      data-testid="delete-guardrail-modal"
    >
      <ModalHeader title="Delete guardrails deployment?" />
      <ModalBody>
        <p>
          The deployment <strong>{deployment.name}</strong> and its running guardrails server will
          be removed from namespace <strong>{deployment.namespace}</strong>. Any applications
          routing through this guardrail will lose its protection.
        </p>
        {error && (
          <Alert
            variant="danger"
            title="Failed to delete deployment"
            isInline
            className="pf-v6-u-mt-md"
          >
            {error.message}
          </Alert>
        )}
      </ModalBody>
      <ModalFooter>
        <Button
          variant="danger"
          onClick={onSubmit}
          isDisabled={isSubmitting}
          data-testid="delete-guardrail-submit-button"
        >
          Delete
        </Button>
        <Button variant="link" onClick={onClose} data-testid="delete-guardrail-cancel-button">
          Cancel
        </Button>
      </ModalFooter>
    </Modal>
  );
};

export default DeleteGuardrailModal;
