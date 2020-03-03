// Code generated by go-swagger; DO NOT EDIT.

package recurring_tasks

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"

	strfmt "github.com/go-openapi/strfmt"

	"github.com/lloydmeta/tasques/models"
)

// DeleteExistingRecurringTaskReader is a Reader for the DeleteExistingRecurringTask structure.
type DeleteExistingRecurringTaskReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *DeleteExistingRecurringTaskReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewDeleteExistingRecurringTaskOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 404:
		result := NewDeleteExistingRecurringTaskNotFound()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result

	default:
		return nil, runtime.NewAPIError("unknown error", response, response.Code())
	}
}

// NewDeleteExistingRecurringTaskOK creates a DeleteExistingRecurringTaskOK with default headers values
func NewDeleteExistingRecurringTaskOK() *DeleteExistingRecurringTaskOK {
	return &DeleteExistingRecurringTaskOK{}
}

/*DeleteExistingRecurringTaskOK handles this case with default header values.

OK
*/
type DeleteExistingRecurringTaskOK struct {
	Payload *models.RecurringTask
}

func (o *DeleteExistingRecurringTaskOK) Error() string {
	return fmt.Sprintf("[DELETE /recurring_tasques/{id}][%d] deleteExistingRecurringTaskOK  %+v", 200, o.Payload)
}

func (o *DeleteExistingRecurringTaskOK) GetPayload() *models.RecurringTask {
	return o.Payload
}

func (o *DeleteExistingRecurringTaskOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.RecurringTask)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewDeleteExistingRecurringTaskNotFound creates a DeleteExistingRecurringTaskNotFound with default headers values
func NewDeleteExistingRecurringTaskNotFound() *DeleteExistingRecurringTaskNotFound {
	return &DeleteExistingRecurringTaskNotFound{}
}

/*DeleteExistingRecurringTaskNotFound handles this case with default header values.

Recurring Task does not exist
*/
type DeleteExistingRecurringTaskNotFound struct {
	Payload *models.CommonBody
}

func (o *DeleteExistingRecurringTaskNotFound) Error() string {
	return fmt.Sprintf("[DELETE /recurring_tasques/{id}][%d] deleteExistingRecurringTaskNotFound  %+v", 404, o.Payload)
}

func (o *DeleteExistingRecurringTaskNotFound) GetPayload() *models.CommonBody {
	return o.Payload
}

func (o *DeleteExistingRecurringTaskNotFound) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.CommonBody)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}