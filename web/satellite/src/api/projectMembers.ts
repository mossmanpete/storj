// Copyright (C) 2018 Storj Labs, Inc.
// See LICENSE for copying information.

import apollo from '@/utils/apolloManager';
import gql from 'graphql-tag';

// Performs graqhQL request.
// Throws an exception if error occurs
export async function addProjectMembers(projectID: string, emails: string[]): Promise<RequestResponse<null>> {
	let result: RequestResponse<null> = {
        errorMessage: '',
        isSuccess: false,
        data: null
	};
	
	try {
		let response: any = await apollo.mutate(
			{
				mutation: gql(`
                mutation {
                    addProjectMembers(
                        projectID: "${projectID}",
                        email: "${emails}"
                    ) {id}
                }`
				),
				fetchPolicy: 'no-cache',
			}
		);

		if (response.errors) {
            result.errorMessage = response.errors[0].message;
        } else {
			result.isSuccess = true;
        }
	} catch (e) {
		result.errorMessage = e.message;
	}

	return result;
}

// Performs graqhQL request.
// Throws an exception if error occurs
export async function deleteProjectMembers(projectID: string, emails: string[]): Promise<RequestResponse<null>> {
	let result: RequestResponse<null> = {
        errorMessage: '',
        isSuccess: false,
        data: null
	};

	try {
		let response: any = await apollo.mutate(
			{
				mutation: gql(`
                mutation {
                    deleteProjectMembers(
                        projectID: "${projectID}",
                        email: "${emails}"
                    ) {id}
                }`
				),
				fetchPolicy: 'no-cache',
			}
		);

		if (response.errors) {
            result.errorMessage = response.errors[0].message;
        } else {
            result.isSuccess = true;
        }
	} catch (e) {
		result.errorMessage = e.message;
	}

	return result;
}

// Performs graqhQL request.
// Throws an exception if error occurs
export async function fetchProjectMembers(projectID: string, limit: string, offset: string): Promise<RequestResponse<TeamMemberModel[]>> {
	let result: RequestResponse<TeamMemberModel[]> = {
        errorMessage: '',
        isSuccess: false,
        data: []
	};

	try {
		let response: any = await apollo.query(
			{
				query: gql(`
					query {
						project(
							id: "${projectID}",
						) {
							members(limit: ${limit}, offset: ${offset}) {
								user {
									id,
									firstName,
									lastName,
									email
								},
								joinedAt
							}
						}
					}`
				),
				fetchPolicy: 'no-cache',
			}
		);

		if (response.errors) {
            result.errorMessage = response.errors[0].message;
        } else {
            result.isSuccess = true;
            result.data = response.data.project.members;
        }
	} catch (e) {
		result.errorMessage = e.message;
	}

	return result;
}
