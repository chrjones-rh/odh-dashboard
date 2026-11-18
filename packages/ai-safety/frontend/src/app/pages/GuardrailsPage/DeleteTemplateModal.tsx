import React from 'react';
import { Alert, Button, Modal, ModalBody, ModalFooter, ModalHeader } from '@patternfly/react-core';
import { useNotification } from '~/app/hooks/useNotification';
import { deleteGuardrailTemplate } from '~/app/api/guardrails';
import { GuardrailTemplate } from '~/app/types';

type DeleteTemplateModalProps = {
  template: GuardrailTemplate | null;
  onClose: () => void;
  onDeleted: () => void;
};

const DeleteTemplateModal: React.FC<DeleteTemplateModalProps> = ({
  template,
  onClose,
  onDeleted,
}) => {
  const notification = useNotification();
  const [isSubmitting, setIsSubmitting] = React.useState(false);
  const [error, setError] = React.useState<Error | null>(null);

  React.useEffect(() => {
    if (template) {
      setError(null);
    }
  }, [template]);

  const onSubmit = React.useCallback(async () => {
    if (!template) {
      return;
    }
    setIsSubmitting(true);
    setError(null);
    try {
      await deleteGuardrailTemplate('', template.namespace, template.name);
      notification.success(
        'Template deleted',
        `Guardrail template ${template.name} was deleted from namespace ${template.namespace}.`,
      );
      onDeleted();
    } catch (e) {
      setError(e instanceof Error ? e : new Error(String(e)));
    } finally {
      setIsSubmitting(false);
    }
  }, [template, notification, onDeleted]);

  if (!template) {
    return null;
  }

  return (
    <Modal
      isOpen
      onClose={onClose}
      variant="small"
      aria-label="Delete guardrail template"
      data-testid="delete-template-modal"
    >
      <ModalHeader title="Delete guardrail template?" />
      <ModalBody>
        <p>
          The template <strong>{template.name}</strong> will be removed from namespace{' '}
          <strong>{template.namespace}</strong>. Deployments already created from it keep their own
          copy of the configuration and are not affected.
        </p>
        {error && (
          <Alert
            variant="danger"
            title="Failed to delete template"
            isInline
            className="pf-v6-u-mt-md"
            data-testid="delete-template-error"
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
          data-testid="delete-template-submit-button"
        >
          Delete
        </Button>
        <Button variant="link" onClick={onClose} data-testid="delete-template-cancel-button">
          Cancel
        </Button>
      </ModalFooter>
    </Modal>
  );
};

export default DeleteTemplateModal;
