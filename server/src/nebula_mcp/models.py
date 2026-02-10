"""Pydantic models for Nebula MCP."""

# Standard Library
from datetime import datetime
from typing import Self

# Third-Party
from pydantic import (
    BaseModel,
    ConfigDict,
    Field,
    StrictInt,
    field_validator,
    model_validator,
)


# --- Core Input Models ---
class CreateEntityInput(BaseModel):
    """Input payload for creating an entity."""

    name: str = Field(..., description="Display name for the entity")
    type: str = Field(..., description="Entity type name")
    status: str = Field(..., description="Status name")
    scopes: list[str] = Field(..., description="Privacy scope names")
    tags: list[str] = Field(default_factory=list, description="Kebab-case tags")
    metadata: dict = Field(
        default_factory=dict, description="Flexible metadata payload"
    )
    vault_file_path: str | None = Field(default=None, description="Source vault path")


# --- Shared Metadata Models ---


class ContextSegment(BaseModel):
    """A single scoped context segment."""

    text: str
    scopes: list[str]


class BaseMetadata(BaseModel):
    """Base metadata shared across entity types."""

    # Allow extra keys to support schema evolution without blocking writes
    model_config = ConfigDict(extra="allow")

    description: str | None = None
    urls: list[str] | None = None
    aliases: list[str] | None = None
    context_segments: list[ContextSegment] | None = None


# --- Type-Specific Metadata Models ---


class PersonMetadata(BaseMetadata):
    """Person metadata with optional structured name and birth fields."""

    first_name: str | None = None
    second_name: str | None = None
    last_name: str | None = None

    birth_year: StrictInt | None = None
    birth_month: StrictInt | None = None
    birth_day: StrictInt | None = None

    location: str | None = None
    uni: str | None = None
    relation: str | None = None
    contact: dict | None = None

    @field_validator("birth_month")
    @classmethod
    def _month_range(cls, v: StrictInt | None) -> StrictInt | None:
        """Validate that birth month is within 1-12 when provided."""

        if v is None:
            return v
        if v < 1 or v > 12:
            raise ValueError("Birth month out of range")
        return v

    @field_validator("birth_day")
    @classmethod
    def _day_range(cls, v: StrictInt | None) -> StrictInt | None:
        """Validate that birth day is within 1-31 when provided."""

        if v is None:
            return v
        if v < 1 or v > 31:
            raise ValueError("Birth day out of range")
        return v

    @model_validator(mode="after")
    def _validate_date_combo(self) -> Self:
        """Validate day ranges for the given month, including leap years."""

        if self.birth_month is not None and self.birth_day is not None:
            max_day = 31
            if self.birth_month in {4, 6, 9, 11}:
                max_day = 30
            elif self.birth_month == 2:
                if self.birth_year and (
                    self.birth_year % 400 == 0
                    or (self.birth_year % 4 == 0 and self.birth_year % 100 != 0)
                ):
                    max_day = 29
                else:
                    max_day = 28
            if self.birth_day > max_day:
                raise ValueError("Birth day invalid for birth month")
        return self


class ProjectMetadata(BaseMetadata):
    """Lightweight project metadata."""

    repository: str | None = None
    tech_stack: list[str] | None = None
    status_note: str | None = None
    start_date: str | None = None


class ToolMetadata(BaseMetadata):
    """Lightweight tool metadata."""

    vendor: str | None = None
    license: str | None = None
    category: str | None = None


class OrganizationMetadata(BaseMetadata):
    """Lightweight organization metadata."""

    industry: str | None = None
    location: str | None = None
    website: str | None = None


class CourseMetadata(BaseMetadata):
    """Lightweight course metadata."""

    institution: str | None = None
    term: str | None = None
    instructor: str | None = None


class IdeaMetadata(BaseMetadata):
    """Lightweight idea metadata."""

    stage: str | None = None
    priority: str | None = None


class FrameworkMetadata(BaseMetadata):
    """Lightweight framework metadata."""

    language: str | None = None
    version: str | None = None


class PaperMetadata(BaseMetadata):
    """Lightweight paper metadata."""

    authors: list[str] | None = None
    year: StrictInt | None = None
    venue: str | None = None


class UniversityMetadata(BaseMetadata):
    """Lightweight university metadata."""

    country: str | None = None
    city: str | None = None


# --- Metadata Validation Helpers ---


def validate_entity_metadata(entity_type: str, metadata: dict) -> dict:
    """Validate metadata by entity type and return a normalized dict.

    Args:
        entity_type: Type name to select validation model.
        metadata: Raw metadata dict to validate.

    Returns:
        Normalized metadata dict with None values excluded.
    """

    type_map: dict[str, type[BaseMetadata]] = {
        "person": PersonMetadata,
        "project": ProjectMetadata,
        "tool": ToolMetadata,
        "organization": OrganizationMetadata,
        "course": CourseMetadata,
        "idea": IdeaMetadata,
        "framework": FrameworkMetadata,
        "paper": PaperMetadata,
        "university": UniversityMetadata,
    }

    model_cls = type_map.get(entity_type, BaseMetadata)
    model = model_cls.model_validate(metadata or {})
    return model.model_dump(exclude_none=True)


# --- Approval Workflow Models ---


class CreateApprovalRequestInput(BaseModel):
    """Input payload for creating an approval request."""

    request_type: str = Field(..., description="Type of action (e.g., create_entity)")
    change_details: dict = Field(..., description="Full payload of requested change")
    job_id: str | None = Field(default=None, description="Optional related job ID")


class ApproveRequestInput(BaseModel):
    """Input payload for approving a request."""

    approval_id: str = Field(..., description="UUID of approval request")
    reviewed_by: str = Field(..., description="UUID of approving entity")


class RejectRequestInput(BaseModel):
    """Input payload for rejecting a request."""

    approval_id: str = Field(..., description="UUID of approval request")
    reviewed_by: str = Field(..., description="UUID of rejecting entity")
    review_notes: str = Field(..., description="Reason for rejection")


class GetApprovalDiffInput(BaseModel):
    """Input payload for approval diff."""

    approval_id: str = Field(..., description="UUID of approval request")


class BulkImportInput(BaseModel):
    """Input payload for bulk imports."""

    format: str = Field(default="json", description="json or csv")
    data: str | None = Field(default=None, description="CSV content when format=csv")
    items: list[dict] | None = Field(default=None, description="JSON item list")
    defaults: dict | None = Field(default=None, description="Defaults for items")


# --- Entity Input Models ---


class GetEntityInput(BaseModel):
    """Input payload for retrieving an entity."""

    entity_id: str = Field(..., description="Entity UUID to retrieve")


class QueryEntitiesInput(BaseModel):
    """Input payload for searching entities."""

    type: str | None = Field(default=None, description="Entity type filter")
    tags: list[str] = Field(default_factory=list, description="Tag filters")
    search_text: str | None = Field(default=None, description="Full-text search query")
    status_category: str = Field(default="active", description="active or archived")
    scopes: list[str] = Field(default_factory=list, description="Privacy scope filters")
    limit: int = Field(default=50, description="Max results to return")
    offset: int = Field(default=0, description="Pagination offset")


class UpdateEntityInput(BaseModel):
    """Input payload for updating an entity."""

    entity_id: str = Field(..., description="Entity UUID to update")
    metadata: dict | None = Field(default=None, description="Updated metadata")
    tags: list[str] | None = Field(default=None, description="Updated tags")
    status: str | None = Field(default=None, description="New status name")
    status_reason: str | None = Field(
        default=None, description="Reason for status change"
    )


class BulkUpdateEntityTagsInput(BaseModel):
    """Input payload for bulk updating entity tags."""

    entity_ids: list[str] = Field(..., description="Entity UUIDs to update")
    tags: list[str] = Field(default_factory=list, description="Tag values")
    op: str = Field(default="add", description="add, remove, or set")


class BulkUpdateEntityScopesInput(BaseModel):
    """Input payload for bulk updating entity scopes."""

    entity_ids: list[str] = Field(..., description="Entity UUIDs to update")
    scopes: list[str] = Field(default_factory=list, description="Scope names")
    op: str = Field(default="add", description="add, remove, or set")


class GetEntityHistoryInput(BaseModel):
    """Input payload for listing entity audit history."""

    entity_id: str = Field(..., description="Entity UUID")
    limit: int = Field(default=50, description="Max results to return")
    offset: int = Field(default=0, description="Pagination offset")


class RevertEntityInput(BaseModel):
    """Input payload for reverting entity to a history entry."""

    entity_id: str = Field(..., description="Entity UUID to revert")
    audit_id: str = Field(..., description="Audit log entry UUID")


class QueryAuditLogInput(BaseModel):
    """Input payload for querying audit log entries."""

    table_name: str | None = Field(default=None, description="Table name filter")
    action: str | None = Field(default=None, description="insert, update, delete")
    actor_type: str | None = Field(default=None, description="agent, entity, system")
    actor_id: str | None = Field(default=None, description="Actor UUID")
    record_id: str | None = Field(default=None, description="Record id filter")
    scope_id: str | None = Field(default=None, description="Scope UUID filter")
    limit: int = Field(default=50, description="Max results to return")
    offset: int = Field(default=0, description="Pagination offset")


class SearchEntitiesByMetadataInput(BaseModel):
    """Input payload for searching entities by metadata fields."""

    metadata_query: dict = Field(..., description="JSONB query object")
    limit: int = Field(default=50, description="Max results to return")


# --- Knowledge Input Models ---


class CreateKnowledgeInput(BaseModel):
    """Input payload for creating a knowledge item."""

    title: str = Field(..., description="Knowledge item title")
    url: str | None = Field(default=None, description="Source URL")
    source_type: str = Field(..., description="article, video, paper, tweet, note")
    content: str | None = Field(default=None, description="Full text content")
    scopes: list[str] = Field(..., description="Privacy scope names")
    tags: list[str] = Field(default_factory=list, description="Kebab-case tags")
    metadata: dict = Field(default_factory=dict, description="Additional metadata")


class QueryKnowledgeInput(BaseModel):
    """Input payload for searching knowledge items."""

    source_type: str | None = Field(default=None, description="Filter by source type")
    tags: list[str] = Field(default_factory=list, description="Tag filters")
    search_text: str | None = Field(default=None, description="Full-text search query")
    scopes: list[str] = Field(default_factory=list, description="Privacy scope filters")
    limit: int = Field(default=50, description="Max results to return")
    offset: int = Field(default=0, description="Pagination offset")


class LinkKnowledgeInput(BaseModel):
    """Input payload for linking knowledge to entity."""

    knowledge_id: str = Field(..., description="Knowledge item UUID")
    entity_id: str = Field(..., description="Entity UUID")
    relationship_type: str = Field(..., description="about, mentions, created-by")


# --- Log Input Models ---


class CreateLogInput(BaseModel):
    """Input payload for creating a log entry."""

    log_type: str = Field(..., description="Log type name")
    timestamp: datetime | None = Field(default=None, description="Timestamp for log")
    value: dict = Field(default_factory=dict, description="Log value payload")
    status: str = Field(default="active", description="Status name")
    tags: list[str] = Field(default_factory=list, description="Kebab-case tags")
    metadata: dict = Field(default_factory=dict, description="Additional metadata")


class GetLogInput(BaseModel):
    """Input payload for retrieving a log entry."""

    log_id: str = Field(..., description="Log UUID")


class QueryLogsInput(BaseModel):
    """Input payload for querying log entries."""

    log_type: str | None = Field(default=None, description="Filter by log type")
    tags: list[str] = Field(default_factory=list, description="Tag filters")
    status_category: str = Field(default="active", description="active or archived")
    limit: int = Field(default=50, description="Max results to return")
    offset: int = Field(default=0, description="Pagination offset")


class UpdateLogInput(BaseModel):
    """Input payload for updating a log entry."""

    id: str = Field(..., description="Log UUID")
    log_type: str | None = Field(default=None, description="Log type name")
    timestamp: datetime | None = Field(default=None, description="Timestamp")
    value: dict | None = Field(default=None, description="Updated value payload")
    status: str | None = Field(default=None, description="Status name")
    tags: list[str] | None = Field(default=None, description="Tags")
    metadata: dict | None = Field(default=None, description="Metadata")


# --- Relationship Input Models ---


class CreateRelationshipInput(BaseModel):
    """Input payload for creating a relationship."""

    source_type: str = Field(
        ..., description="entity, knowledge, log, job, agent, file, protocol"
    )
    source_id: str = Field(..., description="Source item UUID")
    target_type: str = Field(
        ..., description="entity, knowledge, log, job, agent, file, protocol"
    )
    target_id: str = Field(..., description="Target item UUID")
    relationship_type: str = Field(..., description="Relationship type name")
    properties: dict = Field(default_factory=dict, description="Additional properties")


class GetRelationshipsInput(BaseModel):
    """Input payload for retrieving relationships."""

    source_type: str = Field(
        ..., description="entity, knowledge, log, job, agent, file, protocol"
    )
    source_id: str = Field(..., description="Source item UUID")
    relationship_type: str | None = Field(
        default=None, description="Filter by relationship type"
    )
    direction: str = Field(default="both", description="outgoing, incoming, or both")


class QueryRelationshipsInput(BaseModel):
    """Input payload for searching relationships."""

    source_type: str | None = Field(default=None, description="Filter by source type")
    target_type: str | None = Field(default=None, description="Filter by target type")
    relationship_types: list[str] = Field(
        default_factory=list, description="Filter by relationship types"
    )
    status_category: str = Field(default="active", description="active or archived")
    limit: int = Field(default=50, description="Max results to return")


class UpdateRelationshipInput(BaseModel):
    """Input payload for updating a relationship."""

    relationship_id: str = Field(..., description="Relationship UUID")
    properties: dict | None = Field(default=None, description="Updated properties")
    status: str | None = Field(default=None, description="New status name")


class GraphNeighborsInput(BaseModel):
    """Input payload for graph neighbors."""

    source_type: str = Field(
        ..., description="entity, knowledge, log, job, agent, file, protocol"
    )
    source_id: str = Field(..., description="Source item UUID")
    max_hops: int = Field(default=2, description="Max hop depth")
    limit: int = Field(default=100, description="Max results to return")


class GraphShortestPathInput(BaseModel):
    """Input payload for shortest path search."""

    source_type: str = Field(
        ..., description="entity, knowledge, log, job, agent, file, protocol"
    )
    source_id: str = Field(..., description="Source item UUID")
    target_type: str = Field(
        ..., description="entity, knowledge, log, job, agent, file, protocol"
    )
    target_id: str = Field(..., description="Target item UUID")
    max_hops: int = Field(default=6, description="Max hop depth")


# --- Job Input Models ---


class CreateJobInput(BaseModel):
    """Input payload for creating a job."""

    title: str = Field(..., description="Job title")
    description: str | None = Field(default=None, description="Job description")
    job_type: str | None = Field(default=None, description="Job type classification")
    assigned_to: str | None = Field(default=None, description="Assignee entity UUID")
    agent_id: str | None = Field(default=None, description="Agent UUID")
    priority: str = Field(default="medium", description="low, medium, high, critical")
    parent_job_id: str | None = Field(
        default=None, description="Parent job ID for subtasks"
    )
    due_at: str | None = Field(default=None, description="ISO8601 due date")
    metadata: dict = Field(default_factory=dict, description="Additional metadata")


class GetJobInput(BaseModel):
    """Input payload for retrieving a job."""

    job_id: str = Field(..., description="Job ID (YYYYQ#-NNNN format)")


class QueryJobsInput(BaseModel):
    """Input payload for searching jobs."""

    status_names: list[str] = Field(
        default_factory=list, description="Filter by status names"
    )
    assigned_to: str | None = Field(default=None, description="Filter by assignee UUID")
    agent_id: str | None = Field(default=None, description="Filter by agent UUID")
    priority: str | None = Field(default=None, description="Filter by priority")
    due_before: str | None = Field(
        default=None, description="ISO8601 date for due_at filter"
    )
    due_after: str | None = Field(
        default=None, description="ISO8601 date for due_at filter"
    )
    overdue_only: bool = Field(
        default=False, description="Only overdue incomplete jobs"
    )
    parent_job_id: str | None = Field(default=None, description="Filter by parent job")
    limit: int = Field(default=50, description="Max results to return")


class UpdateJobStatusInput(BaseModel):
    """Input payload for updating job status."""

    job_id: str = Field(..., description="Job ID")
    status: str = Field(..., description="New status name")
    status_reason: str | None = Field(
        default=None, description="Reason for status change"
    )
    completed_at: str | None = Field(
        default=None, description="ISO8601 completion timestamp"
    )


class CreateSubtaskInput(BaseModel):
    """Input payload for creating a subtask."""

    parent_job_id: str = Field(..., description="Parent job ID")
    title: str = Field(..., description="Subtask title")
    description: str | None = Field(default=None, description="Subtask description")
    priority: str = Field(default="medium", description="low, medium, high, critical")
    due_at: str | None = Field(default=None, description="ISO8601 due date")


# --- File Input Models ---


class CreateFileInput(BaseModel):
    """Input payload for creating a file record."""

    filename: str = Field(..., description="File name")
    file_path: str = Field(..., description="Absolute or vault-relative path")
    mime_type: str | None = Field(default=None, description="MIME type")
    size_bytes: int | None = Field(default=None, description="File size in bytes")
    checksum: str | None = Field(default=None, description="Checksum hash")
    status: str = Field(default="active", description="Status name")
    tags: list[str] = Field(default_factory=list, description="File tags")
    metadata: dict = Field(default_factory=dict, description="Additional metadata")


class GetFileInput(BaseModel):
    """Input payload for retrieving a file record."""

    file_id: str = Field(..., description="File UUID")


class QueryFilesInput(BaseModel):
    """Input payload for listing files."""

    tags: list[str] = Field(default_factory=list, description="Tag filters")
    mime_type: str | None = Field(default=None, description="Filter by MIME type")
    status_category: str = Field(default="active", description="active or archived")
    limit: int = Field(default=50, description="Max results to return")
    offset: int = Field(default=0, description="Pagination offset")


class UpdateFileInput(BaseModel):
    """Input payload for updating a file record."""

    file_id: str = Field(..., description="File UUID")
    filename: str | None = Field(default=None, description="File name")
    file_path: str | None = Field(default=None, description="Absolute or vault-relative path")
    mime_type: str | None = Field(default=None, description="MIME type")
    size_bytes: int | None = Field(default=None, description="File size in bytes")
    checksum: str | None = Field(default=None, description="Checksum hash")
    status: str | None = Field(default=None, description="Status name")
    tags: list[str] | None = Field(default=None, description="File tags")
    metadata: dict | None = Field(default=None, description="Additional metadata")


class AttachFileInput(BaseModel):
    """Input payload for attaching a file to another record."""

    file_id: str = Field(..., description="File UUID")
    target_id: str = Field(..., description="Target record id")
    relationship_type: str = Field(default="has-file", description="Relationship type")


# --- Protocol Input Models ---


class GetProtocolInput(BaseModel):
    """Input payload for retrieving a protocol."""

    protocol_name: str = Field(..., description="Protocol name (unique identifier)")


class CreateProtocolInput(BaseModel):
    """Input payload for creating a protocol."""

    name: str = Field(..., description="Protocol name (unique identifier)")
    title: str = Field(..., description="Protocol title")
    version: str | None = Field(default=None, description="Protocol version")
    content: str = Field(..., description="Protocol content")
    protocol_type: str | None = Field(default=None, description="Protocol type")
    applies_to: list[str] = Field(
        default_factory=list, description="Applies-to categories"
    )
    status: str = Field(default="active", description="Status name")
    tags: list[str] = Field(default_factory=list, description="Tags")
    metadata: dict = Field(default_factory=dict, description="Metadata")
    vault_file_path: str | None = Field(default=None, description="Vault file path")


class UpdateProtocolInput(BaseModel):
    """Input payload for updating a protocol."""

    name: str = Field(..., description="Protocol name (unique identifier)")
    title: str | None = Field(default=None, description="Protocol title")
    version: str | None = Field(default=None, description="Protocol version")
    content: str | None = Field(default=None, description="Protocol content")
    protocol_type: str | None = Field(default=None, description="Protocol type")
    applies_to: list[str] | None = Field(default=None, description="Applies-to list")
    status: str | None = Field(default=None, description="Status name")
    tags: list[str] | None = Field(default=None, description="Tags")
    metadata: dict | None = Field(default=None, description="Metadata")
    vault_file_path: str | None = Field(default=None, description="Vault file path")


# --- Agent Input Models ---


class GetAgentInfoInput(BaseModel):
    """Input payload for retrieving agent configuration."""

    name: str = Field(..., description="Agent name to retrieve")


class ListAgentsInput(BaseModel):
    """Input payload for listing agents."""

    status_category: str = Field(default="active", description="active or archived")
